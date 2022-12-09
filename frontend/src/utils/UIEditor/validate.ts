import { DatabaseEdit } from "@/types";

interface ValidateResult {
  isValid: boolean;
  messageList: {
    message: string;
    level: "warning" | "error";
  }[];
}

export const validateDatabaseEdit = (
  databaseEdit: DatabaseEdit
): ValidateResult => {
  const validateResult: ValidateResult = {
    isValid: true,
    messageList: [],
  };

  for (const createTableContext of databaseEdit.createTableList) {
    if (createTableContext.addColumnList.length === 0) {
      validateResult.isValid = false;
      validateResult.messageList.push({
        message: `Table ${createTableContext.name} should has at least one column.`,
        level: "error",
      });
    }
  }

  return validateResult;
};
