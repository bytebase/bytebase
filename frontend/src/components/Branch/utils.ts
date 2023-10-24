import { useCurrentUserV1 } from "@/store";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";

export const generateForkedBranchName = (branch: SchemaDesign): string => {
  const currentUser = useCurrentUserV1();
  const parentBranchName = branch.title;
  const branchName =
    `${currentUser.value.title}/${parentBranchName}-draft`.replaceAll(" ", "-");
  return branchName;
};

export const validateBranchName = (branchName: string): boolean => {
  const regex = /^[a-zA-Z][a-zA-Z0-9-_/]+$/;
  return regex.test(branchName);
};

export const wildcardToRegex = (wildcard: string): RegExp => {
  return new RegExp(`^${wildcard.split("*").join(".*")}$`);
};
