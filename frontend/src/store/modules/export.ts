import { useSQLStore } from "@/store";
import type { ExportRequest } from "@/types/proto/api/v1alpha/sql_service";

export const useExportData = () => {
  const sqlStore = useSQLStore();

  const exportData = async (params: ExportRequest) => {
    const { content } = await sqlStore.exportData(params);
    return content;
  };

  return {
    exportData,
  };
};
