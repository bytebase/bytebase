import {
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { normalizeAgentDiagnosticCapability } from "./diagnosticCapability";
import type { AgentDiagnosticContext } from "./diagnosticHelp";
import type { AgentResourceReference } from "./types";

export function getPlanCheckDiagnosticContext({
  resource,
  result,
  sourceId,
  specId,
  sql,
  facts = [],
}: {
  resource: AgentResourceReference;
  result: PlanCheckRun_Result;
  sourceId: string;
  specId?: string;
  sql?: AgentDiagnosticContext["sql"];
  facts?: AgentDiagnosticContext["facts"];
}): AgentDiagnosticContext | undefined {
  if (
    (result.status !== Advice_Level.ERROR &&
      result.status !== Advice_Level.WARNING) ||
    !sourceId.trim() ||
    (!result.title.trim() && !result.content.trim()) ||
    !normalizeAgentDiagnosticCapability({ kind: "diagnostic", resource })
  )
    return undefined;
  const report =
    result.report.case === "sqlReviewReport" ? result.report.value : undefined;
  const start = report?.startPosition;
  const end = report?.endPosition;
  const sha256 = sql?.source === "draft" ? sql.sha256 : result.sheetSha256;
  return {
    resource,
    kind: "sql-check",
    sourceId,
    summary: result.title || result.content,
    detail: result.content,
    sql,
    ...(sha256 && /^[a-f0-9]{64}$/i.test(sha256)
      ? {
          sqlFixSource: {
            ...(specId ? { specId } : {}),
            sha256,
            ...(result.target ? { target: result.target } : {}),
          },
        }
      : {}),
    facts: [
      { label: "Severity", value: Advice_Level[result.status] },
      { label: "Code", value: String(result.code) },
      { label: "Type", value: PlanCheckRun_Result_Type[result.type] },
      ...(result.target ? [{ label: "Target", value: result.target }] : []),
      ...(result.sheetSha256
        ? [{ label: "Sheet SHA256", value: result.sheetSha256 }]
        : []),
      ...(start
        ? [
            {
              label: "SQL range",
              value: `${start.line}:${start.column}${end ? `-${end.line}:${end.column}` : ""}`,
            },
          ]
        : []),
      ...(!sql
        ? [{ label: "SQL source", value: "Unavailable for this result" }]
        : []),
      ...facts,
    ],
  };
}
