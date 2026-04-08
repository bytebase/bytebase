import * as child_process from "child_process";
import * as fs from "fs";
import * as net from "net";
import * as os from "os";
import * as path from "path";

import { BytebaseApiClient } from "./api-client";

const PID_FILE = "/tmp/bytebase-e2e-pid";
const DEFAULT_PORT = 18234;
const DEFAULT_TIMEOUT = 60000;
const ADMIN_EMAIL = "e2e-admin@bytebase.com";
const ADMIN_PASSWORD = "e2e-password-123";

let serverProcess: child_process.ChildProcess | undefined;
let tempDir: string | undefined;

export function cleanupOrphans(): void {
  if (!fs.existsSync(PID_FILE)) return;
  const content = fs.readFileSync(PID_FILE, "utf-8").trim().split("\n");
  const pid = parseInt(content[0], 10);
  const dir = content[1];
  try {
    process.kill(-pid, "SIGTERM"); // Kill process group
  } catch {
    /* already dead */
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

export async function findAvailablePort(): Promise<number> {
  let port = DEFAULT_PORT;
  for (let i = 0; i < 100; i++) {
    const mainFree = await checkPort(port);
    const pgFree = await checkPort(port + 2);
    if (mainFree && pgFree) return port;
    port += 4;
  }
  throw new Error("Could not find available port pair for Bytebase");
}

export async function startServer(): Promise<{
  baseURL: string;
  adminEmail: string;
  adminPassword: string;
}> {
  const binPath = process.env.BYTEBASE_BIN ?? "./bytebase-build/bytebase";
  if (!fs.existsSync(binPath)) {
    throw new Error(
      `Bytebase binary not found at ${binPath}. Build it with:\n` +
        `  go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
    );
  }

  const port = await findAvailablePort();
  tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "bytebase-e2e-"));

  const child = child_process.spawn(
    binPath,
    ["--port", String(port), "--data", tempDir, "--demo"],
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
  while (Date.now() < deadline) {
    try {
      const resp = await fetch(`${baseURL}/healthz`);
      if (resp.ok) break;
    } catch {
      /* not ready */
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  if (Date.now() >= deadline) {
    throw new Error(
      `Bytebase server did not become healthy within ${timeout}ms`
    );
  }

  // Phase 2: Retry signup until success
  const api = new BytebaseApiClient({ baseURL });
  while (Date.now() < deadline) {
    try {
      await api.signup(ADMIN_EMAIL, ADMIN_PASSWORD, "E2E Admin");
      break;
    } catch {
      /* not ready */
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  if (Date.now() >= deadline) {
    throw new Error(
      "Failed to create admin account. Server may not be fully initialized."
    );
  }

  return { baseURL, adminEmail: ADMIN_EMAIL, adminPassword: ADMIN_PASSWORD };
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
