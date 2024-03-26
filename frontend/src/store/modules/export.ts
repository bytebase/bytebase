import { useSQLStore } from "@/store";
import type { ExportFormat } from "@/types/proto/v1/common";
import { extractDatabaseResourceName } from "@/utils";

export type ExportDataParams = {
  format: ExportFormat;
  statement: string;
  limit: number;
  database: string; // instances/{instance}/databases/{database}
  instance: string; // instances/{instance}
  admin?: boolean;
  password?: string;
};

export const useExportData = () => {
  const sqlStore = useSQLStore();

  const exportData = async (params: ExportDataParams) => {
    const { content } = await sqlStore.exportData({
      name: params.database,
      connectionDatabase: extractDatabaseResourceName(params.database)
        .databaseName, // deprecated field, remove me later
      statement: params.statement,
      limit: params.limit,
      format: params.format,
      admin: params.admin ?? false,
      password: params.password ?? "",
    });

    return content;
  };

  return {
    exportData,
  };
};
