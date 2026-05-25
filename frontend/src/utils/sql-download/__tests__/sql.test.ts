// sql.test.ts

import { existsSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { serializeSQL } from "../formats/sql";
import { FIXTURES } from "./fixtures";
import { expectGolden } from "./helpers";

const goldensRoot = resolve(
  dirname(fileURLToPath(import.meta.url)),
  "goldens/sql"
);

const ENGINES: Array<{ key: string; engine: Engine }> = [
  { key: "mysql", engine: Engine.MYSQL },
  { key: "postgres", engine: Engine.POSTGRES },
  { key: "tidb", engine: Engine.TIDB },
  { key: "clickhouse", engine: Engine.CLICKHOUSE },
  { key: "mssql", engine: Engine.MSSQL },
  { key: "oracle", engine: Engine.ORACLE },
  { key: "snowflake", engine: Engine.SNOWFLAKE },
  { key: "sqlite", engine: Engine.SQLITE },
  { key: "redshift", engine: Engine.REDSHIFT },
  { key: "mariadb", engine: Engine.MARIADB },
  { key: "oceanbase", engine: Engine.OCEANBASE },
  { key: "spanner", engine: Engine.SPANNER },
];

describe("serializeSQL byte-equal goldens", () => {
  for (const id of Object.keys(FIXTURES)) {
    for (const { key, engine } of ENGINES) {
      const fileName = `${id}.${key}.sql`;
      const path = resolve(goldensRoot, fileName);
      if (!existsSync(path)) {
        // Backend skipped this engine/fixture combo (e.g. zero columns →
        // SQLStatementPrefix errors) and produced no golden. Skip on the TS
        // side too. Use it.skip to keep the test report informative.
        it.skip(`${id} on ${key} (backend skipped)`, () => {});
        continue;
      }
      it(`${id} on ${key}`, () => {
        const out = serializeSQL(FIXTURES[id], engine);
        expectGolden(out, "sql", fileName);
      });
    }
  }
});

describe("serializeSQL engine guard", () => {
  it("throws UnsupportedFormat for unknown engine", () => {
    expect(() =>
      serializeSQL(FIXTURES.ascii_basic, Engine.ENGINE_UNSPECIFIED)
    ).toThrow(/UnsupportedFormat|engine/i);
  });
});
