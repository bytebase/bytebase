import * as fs from "fs";
import * as path from "path";
import { BytebaseApiClient } from "./api-client";

const ENV_FILE = path.join(__dirname, "../../.e2e-env.json");

export interface TestEnv {
  baseURL: string;
  adminEmail: string;
  adminPassword?: string;
  mode: "local" | "new";
  project: string;
  instance: string;
  instanceId: string;
  database: string;
  databaseId: string;
}

type SerializedTestEnv = Omit<TestEnv, "api">;

export function detectMode(): "local" | "new" {
  return process.env.BYTEBASE_URL ? "local" : "new";
}

export function getBaseURL(): string {
  return process.env.BYTEBASE_URL ?? `http://localhost:${getPort()}`;
}

export function getPort(): number {
  return 18234;
}

export function saveTestEnv(env: TestEnv): void {
  const serialized: SerializedTestEnv = { ...env };
  delete (serialized as Record<string, unknown>)["api"];
  fs.writeFileSync(ENV_FILE, JSON.stringify(serialized, null, 2));
}

export function loadTestEnv(): TestEnv & { api: BytebaseApiClient } {
  if (!fs.existsSync(ENV_FILE)) {
    throw new Error(
      ".e2e-env.json not found. Run the setup project first (npx playwright test)."
    );
  }
  const raw: SerializedTestEnv = JSON.parse(fs.readFileSync(ENV_FILE, "utf-8"));
  const api = new BytebaseApiClient({
    baseURL: raw.baseURL,
    credentials: raw.adminPassword
      ? { email: raw.adminEmail, password: raw.adminPassword }
      : undefined,
  });
  return { ...raw, api };
}

export function cleanupEnvFile(): void {
  if (fs.existsSync(ENV_FILE)) fs.unlinkSync(ENV_FILE);
}
