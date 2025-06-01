import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import type { BinaryLike } from "node:crypto";
import { defineStore } from "pinia";
import { auditLogServiceClient } from "@/grpcweb";
import type { SearchAuditLogsParams } from "@/types";
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
  const fetchAuditLogs = async (search: SearchAuditLogsParams) => {
    const resp = await auditLogServiceClient.searchAuditLogs({
      parent: search.parent,
      filter: buildFilter(search),
      orderBy: search.order ? `create_time ${search.order}` : undefined,
      pageSize: search.pageSize,
      pageToken: search.pageToken,
    });
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
    content: BinaryLike | Blob;
    nextPageToken: string;
  }> => {
    return await auditLogServiceClient.exportAuditLogs({
      parent: search.parent,
      filter: buildFilter(search),
      orderBy: search.order ? `create_time ${search.order}` : undefined,
      format,
      pageSize,
      pageToken,
    });
  };

  return {
    fetchAuditLogs,
    exportAuditLogs,
  };
});
