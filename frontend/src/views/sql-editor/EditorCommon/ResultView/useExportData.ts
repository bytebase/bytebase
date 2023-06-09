import { ref } from "vue";

import { getExportRequestFormat, pushNotification, useSQLStore } from "@/store";
import { extractDatabaseResourceName } from "@/utils";
import dayjs from "dayjs";

export type ExportDataParams = {
  format: "CSV" | "JSON";
  statement: string;
  limit: number;
  database: string; // instances/{instance}/databases/{database}
  instance: string; // instances/{instance}
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
      });

      const blob = new Blob([content], {
        type: params.format === "CSV" ? "text/csv" : "application/json",
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
