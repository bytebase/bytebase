import { projectNamePrefix } from "@/store/modules/v1/common";
import type { SearchAuditLogsParams } from "@/types";
import { AuditLog_Severity } from "@/types/proto-es/v1/audit_log_service_pb";
import {
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  type SearchParams,
} from "@/utils";

export const buildSearchAuditLogParams = (
  searchParams: SearchParams
): SearchAuditLogsParams => {
  const levelScope = getValueFromSearchParams(searchParams, "level");
  const createdTsRange = getTsRangeFromSearchParams(searchParams, "created");

  const params: SearchAuditLogsParams = {
    parent: getValueFromSearchParams(
      searchParams,
      "project",
      projectNamePrefix
    ),
    method: getValueFromSearchParams(searchParams, "method"),
    userEmail: getValueFromSearchParams(searchParams, "actor"),
  };

  if (levelScope) {
    params.level =
      AuditLog_Severity[levelScope as keyof typeof AuditLog_Severity];
  }
  if (createdTsRange) {
    params.createdTsAfter = createdTsRange[0];
    params.createdTsBefore = createdTsRange[1];
  }
  return params;
};
