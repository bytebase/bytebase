import { useSQLStore } from "@/store";
import { ExportFormat } from "@/types/proto/v1/common";
import { extractDatabaseResourceName } from "@/utils";

export type ExportDataParams = {
  format: ExportFormat;
  statement: string;
  limit: number;
  database: string; // instances/{instance}/databases/{database}
  instance: string; // instances/{instance}
  admin?: boolean;
};

export const useExportData = () => {
  const sqlStore = useSQLStore();

  const exportData = async (params: ExportDataParams) => {
    const connectionDatabase = params.database
      ? extractDatabaseResourceName(params.database).database
      : "";

    const { content } = await sqlStore.exportData({
      name: params.instance,
      connectionDatabase,
      statement: params.statement,
      limit: params.limit,
      format: params.format,
      admin: params.admin ?? false,
    });

    return content;
  };

  return {
    exportData,
  };
};
