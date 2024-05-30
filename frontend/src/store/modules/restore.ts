import { useSQLStore } from "@/store";

export type GenerateRestoreSQLParams = {
    name: string; // instances/{instance}/databases/{database}
    statement: string;
    backupDataSource: string; // instances/{instance}/databases/{database} or instances/{instance}/databases/{database}/schemas/{schema} for pg
    backupTable: string;
}

export const useGenerateRestoreSQL = () => {
  const sqlStore = useSQLStore();

  const generateRestoreSQL = async (params: GenerateRestoreSQLParams) => {
    const { statement } = await sqlStore.generateRestoreSQL(params);

    return statement;
  };

  return {
    generateRestoreSQL,
  };
}