import { getExportRequestFormat, useSQLStore } from "@/store";
import { extractDatabaseResourceName } from "@/utils";

export type ExportDataParams = {
  format: "CSV" | "JSON" | "SQL" | "XLSX";
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
      format: getExportRequestFormat(params.format),
      admin: params.admin ?? false,
    });

    return content;
  };

  return {
    exportData,
  };
};
