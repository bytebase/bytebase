import * as fs from "fs";
import * as os from "os";
import * as path from "path";

import type { BytebaseApiClient } from "./api-client";
import {
  ADMIN_EMAIL,
  ADMIN_PASSWORD,
  cleanupOrphans,
  findAvailablePort,
  spawnServer,
  startupTimeoutMs,
  terminateServer,
  waitForAdmin,
} from "./mode-start-new-bytebase";

const PID_FILE = "/tmp/bytebase-e2e-bare-pid";

export interface BareServer {
  baseURL: string;
  adminEmail: string;
  adminPassword: string;
  api: BytebaseApiClient;
  stop: () => Promise<void>;
}

// The hook that calls `startBareServer` needs at least this long, so that a
// slow boot ends in the function's own cleanup and not in a hook timeout.
export const bareServerStartTimeoutMs = () => startupTimeoutMs() + 30_000;

// A second server in first-run state, for a spec the shared one cannot serve:
// its only account is the admin that signed up, it has no project, and its
// sample is unused. It runs on the free plan. The caller owns it and must
// call `stop`; a server left behind by a killed run is removed by the next
// call.
export async function startBareServer(): Promise<BareServer> {
  cleanupOrphans(PID_FILE);
  const port = await findAvailablePort();
  const dataDir = fs.mkdtempSync(path.join(os.tmpdir(), "bytebase-e2e-bare-"));
  const child = spawnServer(port, dataDir, PID_FILE);
  const stop = async () => {
    await terminateServer(child, dataDir);
    fs.rmSync(PID_FILE, { force: true });
  };

  try {
    const baseURL = `http://localhost:${port}`;
    const api = await waitForAdmin(child, baseURL);
    return {
      baseURL,
      adminEmail: ADMIN_EMAIL,
      adminPassword: ADMIN_PASSWORD,
      api,
      stop,
    };
  } catch (error) {
    await stop();
    throw error;
  }
}
