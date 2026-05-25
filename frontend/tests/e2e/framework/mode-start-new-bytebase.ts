import * as child_process from "child_process";
import * as fs from "fs";
import * as net from "net";
import * as os from "os";
import * as path from "path";

import { BytebaseApiClient } from "./api-client";

const PID_FILE = "/tmp/bytebase-e2e-pid";
const DEFAULT_PORT = 18234;
const DEFAULT_TIMEOUT = 300000; // 5 minutes — server startup includes embedded Postgres + migrations + sample-instance bring-up
const ADMIN_EMAIL = "demo@example.com";
const ADMIN_PASSWORD = "12345678"; // NOSONAR: e2e fixture
const ADMIN_TITLE = "Demo";
// DBA fixture used as the second approver by plan-detail approval specs.
// Previously seeded by the demo dump; now provisioned explicitly post-signup.
const DBA_EMAIL = "dba1@example.com";
const DBA_PASSWORD = "12345678"; // NOSONAR: e2e fixture
const DBA_TITLE = "DBA1";
const DBA_ROLE = "roles/workspaceDBA";

let serverProcess: child_process.ChildProcess | undefined;
let tempDir: string | undefined;

// Verify a PID belongs to a bytebase process before sending signals.
// Prevents us from killing an unrelated process if the OS has recycled the PID
// since the last e2e run (e.g. server crashed, PID got reused by another app).
function isBytebaseProcess(pid: number): boolean {
  try {
    const out = child_process
      .execFileSync("ps", ["-p", String(pid), "-o", "command="], {
        encoding: "utf-8",
        stdio: ["ignore", "pipe", "ignore"],
      })
      .trim();
    return out.includes("bytebase");
  } catch {
    return false; // ps exits non-zero if pid doesn't exist
  }
}

export function cleanupOrphans(): void {
  if (!fs.existsSync(PID_FILE)) return;
  const content = fs.readFileSync(PID_FILE, "utf-8").trim().split("\n");
  const pid = parseInt(content[0], 10);
  const dir = content[1];
  if (Number.isFinite(pid) && pid > 0 && isBytebaseProcess(pid)) {
    try {
      process.kill(-pid, "SIGTERM"); // Kill process group
    } catch {
      /* already dead */
    }
  }
  if (dir && fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
  fs.unlinkSync(PID_FILE);
}

function checkPort(port: number): Promise<boolean> {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once("error", () => resolve(false));
    server.once("listening", () => {
      server.close();
      resolve(true);
    });
    server.listen(port, "127.0.0.1");
  });
}

// True when something is actively listening on the port. Inverse of checkPort.
function isPortListening(port: number): Promise<boolean> {
  return new Promise((resolve) => {
    const socket = new net.Socket();
    socket.setTimeout(500);
    socket.once("connect", () => {
      socket.destroy();
      resolve(true);
    });
    socket.once("timeout", () => {
      socket.destroy();
      resolve(false);
    });
    socket.once("error", () => {
      socket.destroy();
      resolve(false);
    });
    socket.connect(port, "127.0.0.1");
  });
}

// Poll until predicate succeeds or deadline passes. On timeout, throws with
// the last underlying error so the cause isn't swallowed.
async function pollUntil(
  predicate: () => Promise<void>,
  deadline: number,
  intervalMs: number,
  timeoutMessage: string
): Promise<void> {
  let lastError: unknown;
  while (Date.now() < deadline) {
    try {
      await predicate();
      return;
    } catch (err) {
      lastError = err;
    }
    await new Promise((r) => setTimeout(r, intervalMs));
  }
  const cause = lastError instanceof Error ? lastError.message : String(lastError ?? "no error captured");
  throw new Error(`${timeoutMessage} Last error: ${cause}`);
}

// Bytebase allocates 4 ports based on the main --port value:
//   PORT     — main HTTP server
//   PORT + 2 — embedded metadata Postgres
//   PORT + 3 — test-sample-instance Postgres (provisioned via SetupSample)
//   PORT + 4 — prod-sample-instance Postgres (provisioned via SetupSample)
// All four must be free before we commit to a base port.
export async function findAvailablePort(): Promise<number> {
  let port = DEFAULT_PORT;
  const offsets = [0, 2, 3, 4];
  for (let i = 0; i < 100; i++) {
    const checks = await Promise.all(offsets.map((o) => checkPort(port + o)));
    if (checks.every((free) => free)) return port;
    port += 5; // skip past all 4 offsets to avoid re-checking conflicting slots
  }
  throw new Error("Could not find available port range for Bytebase");
}

export async function startServer(): Promise<{
  baseURL: string;
  adminEmail: string;
  adminPassword: string;
  hasLicense: boolean;
}> {
  // Check both CWD-relative and repo-root-relative paths
  const candidates = [
    process.env.BYTEBASE_BIN,
    "./bytebase-build/bytebase",
    "../bytebase-build/bytebase",
  ].filter(Boolean) as string[];
  const binPath = candidates.find((p) => fs.existsSync(p));
  if (!binPath) {
    throw new Error(
      `Bytebase binary not found. Build it with:\n` +
        `  pnpm --dir frontend build\n` +
        `  go build -tags embed_frontend -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go\n` +
        `Or set BYTEBASE_BIN to the binary path.`
    );
  }

  const port = await findAvailablePort();
  tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "bytebase-e2e-"));

  const child = child_process.spawn(
    binPath,
    ["--port", String(port), "--data", tempDir],
    {
      detached: true,
      stdio: "ignore",
      env: { ...process.env, PG_URL: "" },
    }
  );

  serverProcess = child;
  child.unref();

  // Write PID file for orphan cleanup
  fs.writeFileSync(PID_FILE, `${child.pid}\n${tempDir}`);

  const baseURL = `http://localhost:${port}`;
  const timeout = parseInt(
    process.env.BYTEBASE_STARTUP_TIMEOUT ?? String(DEFAULT_TIMEOUT),
    10
  );
  const deadline = Date.now() + timeout;

  // Phase 1: Poll /healthz
  await pollUntil(
    async () => {
      const resp = await fetch(`${baseURL}/healthz`);
      if (!resp.ok) throw new Error(`/healthz returned ${resp.status}`);
    },
    deadline,
    500,
    `Bytebase server did not become healthy within ${timeout}ms.`
  );

  // Phase 2: Signup the first admin. Retry while the server finishes
  // migrating its metadata schema (signup returns 4xx until the principal
  // table exists). The first signup becomes workspace admin automatically.
  const api = new BytebaseApiClient({ baseURL });
  await pollUntil(
    () => api.signup(ADMIN_EMAIL, ADMIN_PASSWORD, ADMIN_TITLE),
    deadline,
    500,
    "Failed to signup admin. Server may not be fully initialized."
  );

  // Phase 3: Login to obtain a body token for subsequent API calls.
  // Signup sets cookies only; this api-client uses bearer tokens.
  await api.login(ADMIN_EMAIL, ADMIN_PASSWORD);

  // Phase 3b: Install an enterprise license if one was provided via
  // BYTEBASE_E2E_LICENSE. Specs that exercise gated features (masking,
  // classification) require this; specs read env.hasLicense and skip
  // themselves when it's false. See frontend/tests/e2e/AGENTS.md for how
  // to obtain a dev license.
  const license = process.env.BYTEBASE_E2E_LICENSE?.trim();
  const hasLicense = Boolean(license);
  if (license) {
    await api.uploadLicense(license);
  } else {
    // eslint-disable-next-line no-console
    console.warn(
      "[e2e bootstrap] BYTEBASE_E2E_LICENSE not set — workspace will run on the free plan. Enterprise-gated specs will skip themselves."
    );
  }

  // Phase 3c: Provision the DBA fixture user that plan-detail approval
  // specs use as the second approver. The previous demo dump pre-seeded
  // additional users (dba1/dev1/qa1); now we create only what's actually
  // referenced. Add more fixtures here if new specs need them.
  const { workspace } = await api.getActuatorInfo();
  await api.createUser(DBA_EMAIL, DBA_PASSWORD, DBA_TITLE);
  await api.addWorkspaceRoleMember(workspace, DBA_EMAIL, DBA_ROLE);

  // Phase 4: Provision the sample project + instances on PORT+3 / PORT+4.
  // SetupSample is async on the server but returns immediately; sample
  // Postgres instances come up shortly after.
  await api.setupSample();

  // Wait for both sample instances to register before handing control back —
  // tests rely on discovery finding them via listInstances().
  const requiredInstances = new Set([
    "instances/test-sample-instance",
    "instances/prod-sample-instance",
  ]);
  await pollUntil(
    async () => {
      const { instances } = await api.listInstances();
      const found = new Set(instances?.map((i) => i.name) ?? []);
      const missing = [...requiredInstances].filter((name) => !found.has(name));
      if (missing.length > 0) {
        throw new Error(`Missing sample instances: ${missing.join(", ")}`);
      }
    },
    deadline,
    500,
    "Sample instances did not appear after SetupSample."
  );

  // Phase 5: Wait for the sample Postgres backings to accept TCP connections.
  // listInstances() above only proves the metadata rows exist; the embedded
  // postgres processes come up shortly after. Tests that psql into these
  // instances race against startup without this probe.
  const samplePorts = [port + 3, port + 4];
  await pollUntil(
    async () => {
      const listening = await Promise.all(samplePorts.map(isPortListening));
      const downIdx = listening.findIndex((l) => !l);
      if (downIdx !== -1) {
        throw new Error(`Sample Postgres port ${samplePorts[downIdx]} not accepting connections`);
      }
    },
    deadline,
    500,
    "Sample Postgres instances did not start listening on PORT+3 / PORT+4."
  );

  return { baseURL, adminEmail: ADMIN_EMAIL, adminPassword: ADMIN_PASSWORD, hasLicense };
}

export function stopServer(): void {
  if (serverProcess?.pid) {
    try {
      process.kill(-serverProcess.pid, "SIGTERM");
    } catch {
      /* already dead */
    }
    serverProcess = undefined;
  }
  if (tempDir && fs.existsSync(tempDir)) {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
  if (fs.existsSync(PID_FILE)) {
    fs.unlinkSync(PID_FILE);
  }
}
