import { Branch } from "@/types/proto/v1/branch_service";

export type RebaseSourceType = "BRANCH" | "DATABASE";

export type RebaseBranchValidationState = {
  branch?: Branch;
  conflictSchema?: string;
};
