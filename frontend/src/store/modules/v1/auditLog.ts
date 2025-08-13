import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { defineStore } from "pinia";
import { auditLogServiceClientConnect } from "@/grpcweb";
import type { SearchAuditLogsParams } from "@/types";
import {
  AuditLog_Severity,
  ExportAuditLogsRequestSchema,
  SearchAuditLogsRequestSchema,
} from "@/types/proto-es/v1/audit_log_service_pb";
import { type ExportFormat } from "@/types/proto-es/v1/common_pb";
import { userNamePrefix } from "./common";

dayjs.extend(utc);

const buildFilter = (search: SearchAuditLogsParams): string => {
  const filter: string[] = [];
  if (search.method) {
    filter.push(`method == "${search.method}"`);
  }
  if (search.level) {
    filter.push(`severity == "${AuditLog_Severity[search.level]}"`);
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
  const fetchAuditLogs = async (search: SearchAuditLogsParams) => {
    const request = create(SearchAuditLogsRequestSchema, {
      parent: search.parent,
      filter: buildFilter(search),
      orderBy: search.order ? `create_time ${search.order}` : undefined,
      pageSize: search.pageSize,
      pageToken: search.pageToken,
    });
    const resp = await auditLogServiceClientConnect.searchAuditLogs(request);
    return resp;
  };

  const exportAuditLogs = async ({
    search,
    format,
    pageSize,
    pageToken,
  }: {
    search: SearchAuditLogsParams;
    format: ExportFormat;
    pageSize: number;
    pageToken: string;
  }): Promise<{
    content: Uint8Array;
    nextPageToken: string;
  }> => {
    const request = create(ExportAuditLogsRequestSchema, {
      parent: search.parent,
      filter: buildFilter(search),
      orderBy: search.order ? `create_time ${search.order}` : undefined,
      format,
      pageSize,
      pageToken,
    });
    return await auditLogServiceClientConnect.exportAuditLogs(request);
  };

  return {
    fetchAuditLogs,
    exportAuditLogs,
  };
});
