import * as readline from "readline";
import { BytebaseApiClient } from "./api-client";
import {
  deletePersistedSnapshot,
  loadPersistedSnapshot,
  restoreSnapshot,
} from "./snapshot";

export async function verifyReachable(baseURL: string): Promise<void> {
  try {
    const resp = await fetch(`${baseURL}/healthz`);
    if (!resp.ok) {
      throw new Error(`Bytebase at ${baseURL} returned ${resp.status}`);
    }
  } catch (err) {
    throw new Error(
      `Bytebase server at ${baseURL} is not responding. Is it running?\n` +
        `Start it with: PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug\n` +
        `Original error: ${err}`
    );
  }
}

export async function checkCrashRecovery(
  api: BytebaseApiClient
): Promise<void> {
  const snapshot = loadPersistedSnapshot();
  if (!snapshot || snapshot.status !== "captured") return;

  const isCI = !!process.env.CI;
  let shouldRestore = isCI;

  if (!isCI) {
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    const answer = await new Promise<string>((resolve) => {
      rl.question(
        `Found unrestored snapshot from previous test run (${snapshot.timestamp}). Restore now? [Y/n] `,
        (ans) => {
          rl.close();
          resolve(ans);
        }
      );
    });
    shouldRestore = !answer || answer.toLowerCase() === "y";
  }

  if (shouldRestore) {
    await restoreSnapshot(api, snapshot);
  }
  deletePersistedSnapshot();
}
