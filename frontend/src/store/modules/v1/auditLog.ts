import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { auditLogServiceClient } from "@/grpcweb";
import type { SearchAuditLogsParams } from "@/types";
import type { AuditLog } from "@/types/proto/v1/audit_log_service";
import { userNamePrefix } from "./common";

dayjs.extend(utc);

const buildFilter = (search: SearchAuditLogsParams): string => {
  const filter: string[] = [];
  if (search.parent) {
    filter.push(`parent == "${search.parent}"`);
  }
  if (search.creatorEmail) {
    filter.push(`user == "${userNamePrefix}${search.creatorEmail}"`);
  }
  return filter.join(" && ");
};

export const useAuditLogStore = defineStore("audit_log", () => {
  const auditLogList = reactive(new Map<string, AuditLog[]>());

  const fetchAuditLogs = async (search: SearchAuditLogsParams) => {
    const resp = await auditLogServiceClient.searchAuditLogs({
      filter: buildFilter(search),
      orderBy: search.order ? `create_time ${search.order}` : undefined,
      pageSize: search.pageSize,
      pageToken: search.pageToken,
    });
    for (const auditLog of resp.auditLogs) {
      const list = auditLogList.get(auditLog.resource) || [];
      list.push(auditLog);
      auditLogList.set(auditLog.resource, list);
    }
    return resp;
  };

  return {
    fetchAuditLogs,
  };
});
