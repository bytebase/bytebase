import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { auditLogServiceClient } from "@/grpcweb";
import type { SearchAuditLogsParams } from "@/types";
import type { AuditLog } from "@/types/proto/v1/audit_log_service";
import type { ExportFormat } from "@/types/proto/v1/common";
import { userNamePrefix } from "./common";

dayjs.extend(utc);

const buildFilter = (search: SearchAuditLogsParams): string => {
  const filter: string[] = [];
  if (search.method) {
    filter.push(`method == "${search.method}"`);
  }
  if (search.level) {
    filter.push(`severity == "${search.level}"`);
  }
  if (search.userEmail) {
    filter.push(`user == "${userNamePrefix}${search.userEmail}"`);
  }
  if (search.createdTsAfter) {
    filter.push(
      `create_time >= "${dayjs(search.createdTsAfter).utc().format()}"`
    );
  }
  if (search.createdTsBefore) {
    filter.push(
      `create_time <= "${dayjs(search.createdTsBefore).utc().format()}"`
    );
  }
  return filter.join(" && ");
};

export const useAuditLogStore = defineStore("audit_log", () => {
  const auditLogList = reactive(new Map<string, AuditLog[]>());

  const fetchAuditLogs = async (search: SearchAuditLogsParams) => {
    const resp = await auditLogServiceClient.searchAuditLogs({
      parent: search.parent,
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

  const exportAuditLogs = async (
    search: SearchAuditLogsParams,
    format: ExportFormat
  ) => {
    const { content } = await auditLogServiceClient.exportAuditLogs({
      parent: search.parent,
      filter: buildFilter(search),
      orderBy: search.order ? `create_time ${search.order}` : undefined,
      format,
      // TODO: pagination
      pageSize: 1000,
    });
    return content;
  };

  return {
    fetchAuditLogs,
    exportAuditLogs,
  };
});
