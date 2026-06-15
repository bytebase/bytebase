import { execFileSync } from "child_process";
import type { TestEnv } from "./env";
import type { BytebaseApiClient } from "./api-client";

// Resolve the Postgres port for the database's instance by reading the data
// source from the API. Avoids hardcoding PORT+3 / PORT+4 offsets which would
// break if discovery picks the "test" sample instance instead of "prod".
export async function getInstancePgPort(
  env: TestEnv & { api: BytebaseApiClient },
): Promise<string> {
  const instance = await env.api.getInstance(env.instance);
  const port = instance.dataSources?.[0]?.port;
  if (!port) {
    throw new Error(`Instance ${env.instance} has no data source port`);
  }
  return port;
}

// Execute SQL via psql over the Unix socket on the sample Postgres instance.
// Used for DDL/DML setup and teardown — Bytebase's query API is read-only.
// Callers that interpolate non-constant values MUST validate them first
// (see masking-exemption's assertSafeSqlIdentifier).
export function execSql(dbName: string, port: string, sql: string): void {
  execFileSync(
    "psql",
    [
      "-h", "/tmp",
      "-p", port,
      "-U", "bbsample",
      "-d", dbName,
      "-v", "ON_ERROR_STOP=1",
      "-c", sql,
    ],
    { stdio: "pipe" },
  );
}

// Run a read query and return the raw scalar/tuple output (tuples-only,
// unaligned). Reads as the owner (bbsample), so it reflects the committed
// table state — use it as a positive oracle that a write actually landed,
// instead of inferring success from the absence of a UI error. Same
// interpolation-safety contract as execSql.
export function querySql(dbName: string, port: string, sql: string): string {
  return execFileSync(
    "psql",
    [
      "-h", "/tmp",
      "-p", port,
      "-U", "bbsample",
      "-d", dbName,
      "-v", "ON_ERROR_STOP=1",
      "-tAc", sql,
    ],
    { encoding: "utf-8", stdio: ["ignore", "pipe", "pipe"] },
  ).trim();
}
