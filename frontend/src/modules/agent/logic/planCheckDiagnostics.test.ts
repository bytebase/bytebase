import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { getPlanCheckDiagnosticContext } from "./planCheckDiagnostics";
import type { AgentResourceReference } from "./types";

const resource: AgentResourceReference = {
  kind: "plan",
  name: "projects/example/plans/42",
  project: "projects/example",
  label: "Plan",
};
const result = create(PlanCheckRun_ResultSchema, {
  status: Advice_Level.ERROR,
  code: 123,
  type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
  title: "Rule violation",
  content: "Missing WHERE",
  target: "instances/test/databases/app",
  sheetSha256: "a".repeat(64),
  report: {
    case: "sqlReviewReport",
    value: {
      startPosition: { line: 3, column: 2 },
      endPosition: { line: 4, column: 8 },
    },
  },
});

describe("getPlanCheckDiagnosticContext", () => {
  test.each([Advice_Level.ERROR, Advice_Level.WARNING])(
    "selects explicit problem evidence for severity %s",
    (status) => {
      const sql = { source: "draft" as const, text: "DELETE FROM app;" };
      const context = getPlanCheckDiagnosticContext({
        resource,
        result: { ...result, status },
        sourceId: "draft-check-1",
        sql,
        facts: [{ label: "Spec", value: "change-1" }],
      });
      expect(context).toMatchObject({
        resource,
        kind: "sql-check",
        sourceId: "draft-check-1",
        summary: "Rule violation",
        detail: "Missing WHERE",
        sql,
      });
      expect(context?.facts).toEqual(
        expect.arrayContaining([
          { label: "Severity", value: Advice_Level[status] },
          { label: "Code", value: "123" },
          { label: "Type", value: "STATEMENT_ADVISE" },
          { label: "Target", value: result.target },
          { label: "Sheet SHA256", value: result.sheetSha256 },
          { label: "SQL range", value: "3:2-4:8" },
          { label: "Spec", value: "change-1" },
        ])
      );
    }
  );
  test("keeps missing SQL explicit without inventing persisted source", () => {
    const context = getPlanCheckDiagnosticContext({
      resource,
      result,
      sourceId: `${resource.name}/planCheckRun`,
    });
    expect(context?.sql).toBeUndefined();
    expect(context?.facts).toContainEqual({
      label: "SQL source",
      value: "Unavailable for this result",
    });
  });
  test("identifies saved SQL by the recorded result hash and target", () => {
    const context = getPlanCheckDiagnosticContext({
      resource,
      result,
      sourceId: "saved-check-1",
      specId: "spec-1",
      sql: { source: "saved", text: "SELECT 1;", sha256: "b".repeat(64) },
    });
    expect(context?.sqlFixSource).toEqual({
      specId: "spec-1",
      sha256: result.sheetSha256,
      target: result.target,
    });
  });
  test("identifies draft SQL by its checked bytes, not the result hash", () => {
    const context = getPlanCheckDiagnosticContext({
      resource,
      result,
      sourceId: "draft-check-1",
      specId: "spec-1",
      sql: { source: "draft", text: "SELECT 1;", sha256: "b".repeat(64) },
    });
    expect(context?.sqlFixSource).toEqual({
      specId: "spec-1",
      sha256: "b".repeat(64),
      target: result.target,
    });
  });
  test.each(["", "a".repeat(63), "g".repeat(64), ` ${"a".repeat(64)}`])(
    "keeps saved checks with invalid recorded hash %j diagnose-only",
    (sheetSha256) => {
      const context = getPlanCheckDiagnosticContext({
        resource,
        result: { ...result, sheetSha256 },
        sourceId: "saved-check-1",
      });
      expect(context).toBeDefined();
      expect(context?.sqlFixSource).toBeUndefined();
    }
  );
  test.each([undefined, "", "invalid"])(
    "does not replace missing or invalid draft hash %j with the saved hash",
    (sha256) => {
      const context = getPlanCheckDiagnosticContext({
        resource,
        result,
        sourceId: "draft-check-1",
        sql: { source: "draft", text: "SELECT 1;", sha256 },
      });
      expect(context).toBeDefined();
      expect(context?.sqlFixSource).toBeUndefined();
    }
  );
  test.each([Advice_Level.SUCCESS, Advice_Level.ADVICE_LEVEL_UNSPECIFIED])(
    "suppresses non-problem severity %s",
    (status) => {
      expect(
        getPlanCheckDiagnosticContext({
          resource,
          result: { ...result, status },
          sourceId: "check-1",
        })
      ).toBeUndefined();
    }
  );
  test.each([
    { ...resource, name: "projects/other/plans/42" },
    { ...resource, name: "projects/example/plans/create" },
    { ...resource, name: "projects/example/plans/-" },
  ])("rejects invalid resource $name", (invalid) => {
    expect(
      getPlanCheckDiagnosticContext({
        resource: invalid,
        result,
        sourceId: "check-1",
      })
    ).toBeUndefined();
  });
  test("requires source identity and problem text", () => {
    expect(
      getPlanCheckDiagnosticContext({ resource, result, sourceId: " " })
    ).toBeUndefined();
    expect(
      getPlanCheckDiagnosticContext({
        resource,
        result: { ...result, title: "", content: "" },
        sourceId: "check-1",
      })
    ).toBeUndefined();
  });
});
