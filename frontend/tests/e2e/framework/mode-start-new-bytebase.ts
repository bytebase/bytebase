import * as child_process from "child_process";
import * as fs from "fs";
import * as net from "net";
import * as os from "os";
import * as path from "path";

import { BytebaseApiClient } from "./api-client";

const PID_FILE = "/tmp/bytebase-e2e-pid";
const DEFAULT_PORT = 18234;
const DEFAULT_TIMEOUT = 300000; // 5 minutes — server startup includes embedded Postgres + migrations + sample-instance bring-up
export const ADMIN_EMAIL = "demo@example.com";
export const ADMIN_PASSWORD = "12345678"; // NOSONAR: e2e fixture
const ADMIN_TITLE = "Demo";
const SAMPLE_PROJECT_ID = "project-sample";
const SAMPLE_PROJECT_TITLE = "Sample Project";
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

export function cleanupOrphans(pidFile: string = PID_FILE): void {
  if (!fs.existsSync(pidFile)) return;
  const content = fs.readFileSync(pidFile, "utf-8").trim().split("\n");
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
  fs.unlinkSync(pidFile);
}

function checkPort(port: number): Promise<boolean> {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once("error", () => resolve(false));
    server.once("listening", () => {
      server.close();
      resolve(true);
    });
    // No host, as the server itself binds: macOS lets a 127.0.0.1 bind succeed
    // beside a wildcard listener, which would hand out a running server's port.
    server.listen(port);
  });
}

// True when something is actively listening on the port. The bytebase
// sample-postgres processes are started with `-h ""` (no TCP listen) +
// `-k /tmp` (Unix socket directory), so a TCP probe ALWAYS fails even
// when the postgres is up and answering queries — verified by reading
// the `ps -ef` output of a running e2e instance. Probe the Unix socket
// at /tmp/.s.PGSQL.${port} first; fall back to TCP for any environment
// that does configure TCP (CI containers, future flag changes).
function isPortListening(port: number): Promise<boolean> {
  return new Promise((resolve) => {
    // Try the Unix socket first.
    const unixPath = `/tmp/.s.PGSQL.${port}`;
    const unix = new net.Socket();
    let unixDone = false;
    unix.setTimeout(500);
    const done = (value: boolean) => {
      if (unixDone) return;
      unixDone = true;
      unix.destroy();
      if (value) {
        resolve(true);
        return;
      }
      // Fallback: TCP probe.
      const tcp = new net.Socket();
      tcp.setTimeout(500);
      tcp.once("connect", () => {
        tcp.destroy();
        resolve(true);
      });
      tcp.once("timeout", () => {
        tcp.destroy();
        resolve(false);
      });
      tcp.once("error", () => {
        tcp.destroy();
        resolve(false);
      });
      tcp.connect(port, "127.0.0.1");
    };
    unix.once("connect", () => done(true));
    unix.once("timeout", () => done(false));
    unix.once("error", () => done(false));
    unix.connect(unixPath);
  });
}

// Poll until predicate succeeds or deadline passes. On timeout, throws with
// the last underlying error so the cause isn't swallowed. `giveUp` ends the
// wait early, with the reason it returns, when succeeding has become
// impossible.
async function pollUntil(
  predicate: () => Promise<void>,
  deadline: number,
  intervalMs: number,
  timeoutMessage: string,
  giveUp?: () => string | undefined
): Promise<void> {
  let lastError: unknown;
  while (Date.now() < deadline) {
    const reason = giveUp?.();
    if (reason) throw new Error(reason);
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

// Bytebase allocates 3 ports based on the main --port value:
//   PORT     — main HTTP server
//   PORT + 2 — embedded metadata Postgres
//   PORT + 3 — project-scoped sample Postgres
// All three must be free before we commit to a base port.
export async function findAvailablePort(): Promise<number> {
  let port = DEFAULT_PORT;
  const offsets = [0, 2, 3];
  for (let i = 0; i < 100; i++) {
    const checks = await Promise.all(offsets.map((o) => checkPort(port + o)));
    // The embedded and sample Postgres listen on Unix sockets only, so a TCP
    // bind cannot see another server's.
    const sockets = await Promise.all(
      [2, 3].map((o) => isPortListening(port + o))
    );
    if (checks.every((free) => free) && sockets.every((used) => !used)) {
      return port;
    }
    port += 4; // skip past all 3 offsets to avoid re-checking conflicting slots
  }
  throw new Error("Could not find available port range for Bytebase");
}

function findServerBinary(): string {
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
  return binPath;
}

export function startupTimeoutMs(): number {
  return parseInt(
    process.env.BYTEBASE_STARTUP_TIMEOUT ?? String(DEFAULT_TIMEOUT),
    10
  );
}

// BYTEBASE_E2E_LICENSE wins; otherwise the dev-signed enterprise license that
// frontend/.env.dev commits for dev builds. The dev key only validates on a
// binary built without the `release` tag, which is what the README prescribes.
export function resolveLicense(): string {
  const fromEnv = process.env.BYTEBASE_E2E_LICENSE?.trim();
  if (fromEnv) {
    return fromEnv;
  }
  const envFile = path.resolve(__dirname, "../../../.env.dev");
  if (!fs.existsSync(envFile)) {
    return "";
  }
  for (const line of fs.readFileSync(envFile, "utf8").split("\n")) {
    const match = line.match(/^BB_DEV_ENTERPRISE_LICENSE=(.*)$/);
    if (match) {
      return match[1].trim().replace(/^["']|["']$/g, "");
    }
  }
  return "";
}

// Detached, so the server leads a process group `terminateServer` can signal.
// Its Postgres processes run outside that group and stop when the server
// shuts down. `pidFile` records the server for `cleanupOrphans`.
export function spawnServer(
  port: number,
  dataDir: string,
  pidFile: string = PID_FILE
): child_process.ChildProcess {
  const child = child_process.spawn(
    findServerBinary(),
    ["--port", String(port), "--data", dataDir],
    {
      detached: true,
      stdio: "ignore",
      // The embedded Postgres needs a valid UTF-8 locale: without LANG,
      // initdb rejects the locale and Postgres 17 on macOS then fails with
      // "postmaster became multithreaded during startup".
      env: { LANG: "en_US.UTF-8", ...process.env, PG_URL: "" },
    }
  );
  child.unref();
  fs.writeFileSync(pidFile, `${child.pid}\n${dataDir}`);
  return child;
}

const hasExited = (child: child_process.ChildProcess) =>
  child.exitCode !== null || child.signalCode !== null;

// Waits until `child` serves requests, then signs up the fixture admin, who
// becomes workspace admin as the first account, and logs the client in.
//
// Gives up as soon as `child` has exited. /healthz is asked by port, so
// otherwise a server that died at boot would be polled to the deadline, and
// another server on the same port would be taken for it.
export async function waitForAdmin(
  child: child_process.ChildProcess,
  baseURL: string
): Promise<BytebaseApiClient> {
  const timeout = startupTimeoutMs();
  const exited = () =>
    hasExited(child) ? "Bytebase server exited during startup." : undefined;
  await pollUntil(
    async () => {
      const resp = await fetch(`${baseURL}/healthz`);
      if (!resp.ok) throw new Error(`/healthz returned ${resp.status}`);
    },
    Date.now() + timeout,
    500,
    `Bytebase server did not become healthy within ${timeout}ms.`,
    exited
  );
  const reason = exited();
  if (reason) throw new Error(reason);

  // The schema is migrated before the server listens, so signup needs no
  // retry. It sets cookies only; this client uses the bearer token of a login.
  const api = new BytebaseApiClient({
    baseURL,
    credentials: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD },
  });
  await api.signup(ADMIN_EMAIL, ADMIN_PASSWORD, ADMIN_TITLE);
  await api.login(ADMIN_EMAIL, ADMIN_PASSWORD);
  return api;
}

// Stops `child` with its process group, then removes `dataDir`.
//
// The directory goes only after the server has exited. Deleting it while the
// embedded Postgres is still shutting down makes the checkpointer PANIC
// ("could not fsync pg_xact/0000") and leaves the metadata-Postgres socket
// (/tmp/.s.PGSQL.<PORT+2>) behind, which then trips the next boot's port
// probe. The wait must be asynchronous: Node reaps the child on the event
// loop, so a blocking poll of `kill(pid, 0)` would only ever see a zombie and
// run to its deadline. Bounded (ref'd timers keep the loop alive): a server
// that ignores SIGTERM for 15 s is SIGKILLed as a group and given a short
// grace so the directory is never deleted under a live Postgres.
export async function terminateServer(
  child: child_process.ChildProcess,
  dataDir: string | undefined
): Promise<void> {
  const exited = () => hasExited(child);
  const waitExit = (ms: number) =>
    new Promise<void>((resolve) => {
      if (exited()) {
        resolve();
        return;
      }
      const timer = setTimeout(resolve, ms);
      child.once("exit", () => {
        clearTimeout(timer);
        resolve();
      });
    });
  const signalGroup = (signal: NodeJS.Signals) => {
    try {
      process.kill(-child.pid!, signal);
    } catch {
      /* already dead */
    }
  };
  signalGroup("SIGTERM");
  await waitExit(15_000);
  if (!exited()) {
    signalGroup("SIGKILL");
    await waitExit(5_000);
  }
  if (dataDir && fs.existsSync(dataDir)) {
    fs.rmSync(dataDir, { recursive: true, force: true });
  }
}

export async function startServer(): Promise<{
  baseURL: string;
  adminEmail: string;
  adminPassword: string;
  project: string;
  instance: string;
}> {
  // License is REQUIRED (the suite is enterprise-only). Resolve it BEFORE
  // booting the server, so a missing license fails fast and never orphans a
  // server process.
  const license = resolveLicense();
  if (!license) {
    throw new Error(
      "No enterprise license: set BYTEBASE_E2E_LICENSE, or keep the dev license " +
        "line BB_DEV_ENTERPRISE_LICENSE in frontend/.env.dev. The suite does not " +
        "run on the free plan.",
    );
  }

  const port = await findAvailablePort();
  tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "bytebase-e2e-"));
  const child = spawnServer(port, tempDir);
  serverProcess = child;

  const baseURL = `http://localhost:${port}`;
  const deadline = Date.now() + startupTimeoutMs();
  const api = await waitForAdmin(child, baseURL);

  // Phase 3b: Install the enterprise license. Its presence was verified at
  // the top of startServer (before boot); install it now that the API client
  // is logged in. See frontend/tests/e2e/AGENTS.md for a dev license.
  await api.uploadLicense(license);

  // Phase 3c: Provision the DBA fixture user that plan-detail approval
  // specs use as the second approver. The previous demo dump pre-seeded
  // additional users (dba1/dev1/qa1); now we create only what's actually
  // referenced. Add more fixtures here if new specs need them.
  const { workspace } = await api.getActuatorInfo();
  await api.createUser(DBA_EMAIL, DBA_PASSWORD, DBA_TITLE);
  await api.addWorkspaceRoleMember(workspace, DBA_EMAIL, DBA_ROLE);

  // Phase 4: Create the sample project and provision its instance on PORT+3.
  const sampleProject = await api.createProject(
    SAMPLE_PROJECT_ID,
    SAMPLE_PROJECT_TITLE,
  );
  const sampleInstance = await api.prepareSampleProjectInstance(
    sampleProject.name,
  );

  // Wait for the project-scoped sample instance to be visible before handing
  // control back. Its resource ID is generated by the sample manager.
  await pollUntil(
    async () => {
      const { instances } = await api.listInstances(sampleProject.name);
      if (!instances?.some((instance) => instance.name === sampleInstance.name)) {
        throw new Error(`Missing sample instance: ${sampleInstance.name}`);
      }
    },
    deadline,
    500,
    "Sample instance did not appear after PrepareSampleProjectInstance."
  );

  // Phase 5: Wait for the sample Postgres backings to accept TCP connections.
  // listInstances() above only proves the metadata rows exist; the embedded
  // postgres processes come up shortly after. Tests that psql into these
  // instances race against startup without this probe.
  const samplePorts = [port + 3];
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
    "Sample Postgres instance did not start listening on PORT+3."
  );

  return {
    baseURL,
    adminEmail: ADMIN_EMAIL,
    adminPassword: ADMIN_PASSWORD,
    project: sampleProject.name,
    instance: sampleInstance.name,
  };
}

export async function stopServer(): Promise<void> {
  const child = serverProcess;
  const dataDir = tempDir;
  serverProcess = undefined;
  if (child?.pid) {
    await terminateServer(child, dataDir);
  } else if (dataDir && fs.existsSync(dataDir)) {
    fs.rmSync(dataDir, { recursive: true, force: true });
  }
  if (fs.existsSync(PID_FILE)) {
    fs.unlinkSync(PID_FILE);
  }
}
