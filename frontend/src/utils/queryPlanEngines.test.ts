import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { QueryOption_ExplainFormat } from "@/types/proto-es/v1/sql_service_pb";
import {
  explainQueryOption,
  isVisualizerEngine,
  VISUALIZER_EXPLAIN_FORMATS,
} from "./queryPlanEngines";
import { instanceV1AllowsExplain } from "./v1/instance";

describe("isVisualizerEngine", () => {
  test("covers the engines the viewer has a parser for", () => {
    expect(Object.keys(VISUALIZER_EXPLAIN_FORMATS).map(Number).sort()).toEqual(
      [
        Engine.POSTGRES,
        Engine.MSSQL,
        Engine.SPANNER,
        Engine.MYSQL,
        Engine.MARIADB,
        Engine.TIDB,
        Engine.ORACLE,
        Engine.CLICKHOUSE,
        Engine.TRINO,
        Engine.OCEANBASE,
        Engine.MONGODB,
        Engine.COCKROACHDB,
        Engine.SNOWFLAKE,
        Engine.HIVE,
        Engine.BIGQUERY,
        Engine.REDSHIFT,
        Engine.STARROCKS,
        Engine.DORIS,
        Engine.DATABRICKS,
      ].sort()
    );
    expect(isVisualizerEngine(Engine.POSTGRES)).toBe(true);
    expect(isVisualizerEngine(Engine.MYSQL)).toBe(true);
    expect(isVisualizerEngine(Engine.SNOWFLAKE)).toBe(true);
    expect(isVisualizerEngine(Engine.DATABRICKS)).toBe(true);
    expect(isVisualizerEngine(Engine.CASSANDRA)).toBe(false);
  });

  test("names only engines whose plan the editor can reach", () => {
    // Explain is how a plan is asked for, so a parser for an engine the editor
    // does not explain would never be reached — unless the plan comes back with
    // an ordinary run, which is how BigQuery reports one.
    for (const engine of Object.keys(VISUALIZER_EXPLAIN_FORMATS).map(Number)) {
      if (engine === Engine.BIGQUERY) continue;
      expect(instanceV1AllowsExplain(engine), Engine[engine]).toBe(true);
    }
    expect(instanceV1AllowsExplain(Engine.BIGQUERY)).toBe(false);
  });
});

describe("explainQueryOption", () => {
  test("asks for the format the viewer draws, so a plan takes one call", () => {
    // Without this the engine answers in its own default — text, on
    // PostgreSQL — and drawing it would mean explaining a second time.
    expect(explainQueryOption(Engine.POSTGRES).explainFormat).toBe(
      QueryOption_ExplainFormat.JSON
    );
    expect(explainQueryOption(Engine.MSSQL).explainFormat).toBe(
      QueryOption_ExplainFormat.XML
    );
    expect(explainQueryOption(Engine.SPANNER).explainFormat).toBe(
      QueryOption_ExplainFormat.JSON
    );
    // These plans are text; the viewer draws them because these engines
    // publish nothing else.
    for (const engine of [
      Engine.REDSHIFT,
      Engine.STARROCKS,
      Engine.DORIS,
      Engine.DATABRICKS,
    ]) {
      expect(explainQueryOption(engine).explainFormat).toBe(
        QueryOption_ExplainFormat.TEXT
      );
    }
    // Each remaining engine has one structured plan, which the server spells
    // in that engine's own syntax — or, for Oracle, renders itself.
    for (const engine of [
      Engine.MYSQL,
      Engine.MARIADB,
      Engine.TIDB,
      Engine.ORACLE,
      Engine.CLICKHOUSE,
      Engine.TRINO,
      Engine.OCEANBASE,
      Engine.MONGODB,
      Engine.COCKROACHDB,
      Engine.SNOWFLAKE,
      Engine.HIVE,
    ]) {
      expect(explainQueryOption(engine).explainFormat).toBe(
        QueryOption_ExplainFormat.JSON
      );
    }
  });

  test("leaves BigQuery's explain request alone, since it is a dry run", () => {
    // Asking for the format the viewer draws would refuse the dry run, and the
    // plan it draws arrives with an ordinary run rather than this request.
    expect(explainQueryOption(Engine.BIGQUERY).explainFormat).toBe(
      QueryOption_ExplainFormat.EXPLAIN_FORMAT_UNSPECIFIED
    );
  });

  test("leaves an engine the viewer cannot draw on its own default", () => {
    // What is left are the engines that stated no plan to begin with: CQL has
    // no EXPLAIN, and the document and key-value APIs publish none.
    for (const engine of [
      Engine.CASSANDRA,
      Engine.REDIS,
      Engine.DYNAMODB,
      Engine.COSMOSDB,
      Engine.ELASTICSEARCH,
    ]) {
      expect(explainQueryOption(engine).explainFormat, Engine[engine]).toBe(
        QueryOption_ExplainFormat.EXPLAIN_FORMAT_UNSPECIFIED
      );
    }
  });
});
