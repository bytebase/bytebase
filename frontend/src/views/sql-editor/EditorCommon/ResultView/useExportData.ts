import dayjs from "dayjs";
import { ref } from "vue";
import {
  getExportFileType,
  getExportRequestFormat,
  pushNotification,
  useSQLStore,
} from "@/store";
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
  const isExportingData = ref(false);
  const sqlStore = useSQLStore();

  const exportData = async (params: ExportDataParams) => {
    isExportingData.value = true;

    try {
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

      const blob = new Blob([content], {
        type: getExportFileType(params.format),
      });
      const url = window.URL.createObjectURL(blob);

      const fileExt = params.format.toLowerCase();
      const formattedDateString = dayjs(new Date()).format(
        "YYYY-MM-DDTHH-mm-ss"
      );
      const filename = `export-data-${formattedDateString}`;
      const link = document.createElement("a");
      link.download = `${filename}.${fileExt}`;
      link.href = url;
      link.click();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `Failed to export data`,
        description: JSON.stringify(error),
      });
    } finally {
      isExportingData.value = false;
    }
  };

  return {
    isExportingData,
    exportData,
  };
};
