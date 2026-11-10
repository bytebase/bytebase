import { type ChildProcess, execFileSync, spawn } from "child_process";
import type { TestEnv } from "./env";
import type { BytebaseApiClient } from "./api-client";

// Resolve the Postgres port for the database's instance by reading the data
// source from the API instead of hardcoding the sample Postgres port.
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

// The sample Postgres listens on a Unix socket in /tmp as role bbsample.
function psqlArgs(dbName: string, port: string): string[] {
  return ["-h", "/tmp", "-p", port, "-U", "bbsample", "-d", dbName, "-v", "ON_ERROR_STOP=1"];
}

// Execute SQL via psql over the Unix socket on the sample Postgres instance.
// Used for DDL/DML setup and teardown — Bytebase's query API is read-only.
// Callers that interpolate non-constant values MUST validate them first
// (see masking-exemption's assertSafeSqlIdentifier).
export function execSql(dbName: string, port: string, sql: string): void {
  execFileSync("psql", [...psqlArgs(dbName, port), "-c", sql], { stdio: "pipe" });
}

// Execute a larger SQL script through stdin so it does not hit the operating
// system's command-line length limit.
export function execSqlScript(dbName: string, port: string, sql: string): void {
  execFileSync("psql", psqlArgs(dbName, port), {
    input: sql,
    stdio: ["pipe", "pipe", "pipe"],
  });
}

// Run a read query and return the raw scalar/tuple output (tuples-only,
// unaligned). Reads as the owner (bbsample), so it reflects the committed
// table state — use it as a positive oracle that a write actually landed,
// instead of inferring success from the absence of a UI error. Same
// interpolation-safety contract as execSql.
export function querySql(dbName: string, port: string, sql: string): string {
  return execFileSync("psql", [...psqlArgs(dbName, port), "-tAc", sql], {
    encoding: "utf-8",
    stdio: ["ignore", "pipe", "pipe"],
  }).trim();
}

// A psql session that stays open after running `sql`, for a lock a test must
// hold across steps: the caller ends it by killing the process, and the server
// rolls back whatever the session left open. Its errors go to the run output.
export function spawnPsql(dbName: string, port: string, sql: string): ChildProcess {
  const child = spawn("psql", psqlArgs(dbName, port), { stdio: ["pipe", "ignore", "inherit"] });
  child.stdin?.write(`${sql}\n`);
  return child;
}
