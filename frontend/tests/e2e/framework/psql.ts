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
