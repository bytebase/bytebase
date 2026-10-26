import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { QueryOption_ExplainFormat } from "@/types/proto-es/v1/sql_service_pb";
import {
  explainQueryOption,
  isVisualizerEngine,
  VISUALIZER_EXPLAIN_FORMATS,
} from "./queryPlanEngines";

describe("isVisualizerEngine", () => {
  test("covers the engines the viewer has a parser for", () => {
    expect(Object.keys(VISUALIZER_EXPLAIN_FORMATS).map(Number).sort()).toEqual(
      [Engine.POSTGRES, Engine.MSSQL, Engine.SPANNER].sort()
    );
    expect(isVisualizerEngine(Engine.POSTGRES)).toBe(true);
    expect(isVisualizerEngine(Engine.MYSQL)).toBe(false);
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
  });

  test("leaves an engine the viewer cannot draw on its own default", () => {
    expect(explainQueryOption(Engine.MYSQL).explainFormat).toBe(
      QueryOption_ExplainFormat.EXPLAIN_FORMAT_UNSPECIFIED
    );
    expect(explainQueryOption(Engine.ORACLE).explainFormat).toBe(
      QueryOption_ExplainFormat.EXPLAIN_FORMAT_UNSPECIFIED
    );
  });
});
