import { DatabaseEdit, ValidateResult } from "@/types";

export const validateDatabaseEdit = (
  databaseEdit: DatabaseEdit
): ValidateResult[] => {
  const validateResultList: ValidateResult[] = [];

  for (const createTableContext of databaseEdit.createTableList) {
    if (createTableContext.addColumnList.length === 0) {
      validateResultList.push({
        type: "ERROR",
        message: `Table ${createTableContext.name} should has at least one column.`,
      });
    }
  }

  return validateResultList;
};
