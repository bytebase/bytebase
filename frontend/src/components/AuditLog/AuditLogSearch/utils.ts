import { projectNamePrefix } from "@/store/modules/v1/common";
import type { SearchAuditLogsParams } from "@/types";
import { AuditLog_Severity } from "@/types/proto-es/v1/audit_log_service_pb";
import { getTsRangeFromSearchParams, type SearchParams } from "@/utils";

export const buildSearchAuditLogParams = (
  searchParams: SearchParams
): SearchAuditLogsParams => {
  const { scopes } = searchParams;
  const projectScope = scopes.find((s) => s.id === "project");
  const levelScope = scopes.find((s) => s.id === "level");
  const methodScope = scopes.find((s) => s.id === "method");
  const actorScope = scopes.find((s) => s.id === "actor");
  const createdTsRange = getTsRangeFromSearchParams(searchParams, "created");

  const params: SearchAuditLogsParams = {};
  if (projectScope) {
    params.parent = `${projectNamePrefix}${projectScope.value}`;
  }
  if (levelScope) {
    params.level =
      AuditLog_Severity[levelScope.value as keyof typeof AuditLog_Severity];
  }
  if (methodScope) {
    params.method = methodScope.value;
  }
  if (actorScope) {
    params.userEmail = actorScope.value;
  }
  if (createdTsRange) {
    params.createdTsAfter = createdTsRange[0];
    params.createdTsBefore = createdTsRange[1];
  }
  return params;
};
