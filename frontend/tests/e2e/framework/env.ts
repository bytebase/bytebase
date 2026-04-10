import * as fs from "fs";
import * as path from "path";
import { BytebaseApiClient } from "./api-client";

const ENV_FILE = path.join(__dirname, "../../.e2e-env.json");

export interface TestEnv {
  baseURL: string;
  adminEmail: string;
  adminPassword: string;
  project: string;
  instance: string;
  instanceId: string;
  database: string;
  databaseId: string;
}

export function saveTestEnv(env: TestEnv): void {
  const serialized = { ...env } as Record<string, unknown>;
  delete serialized["api"];
  fs.writeFileSync(ENV_FILE, JSON.stringify(serialized, null, 2));
}

export function loadTestEnv(): TestEnv & { api: BytebaseApiClient } {
  if (!fs.existsSync(ENV_FILE)) {
    throw new Error(
      ".e2e-env.json not found. Run the setup project first (npx playwright test)."
    );
  }
  const raw = JSON.parse(fs.readFileSync(ENV_FILE, "utf-8")) as TestEnv;
  const api = new BytebaseApiClient({
    baseURL: raw.baseURL,
    credentials: { email: raw.adminEmail, password: raw.adminPassword },
  });
  return { ...raw, api };
}

export function cleanupEnvFile(): void {
  if (fs.existsSync(ENV_FILE)) fs.unlinkSync(ENV_FILE);
}
