import type { Status } from "nice-grpc-common";
import type { Branch } from "@/types/proto/v1/branch_service";

export type MergeBranchValidationState = {
  status: Status;
  branch?: Branch;
  errmsg?: string;
};

export type PostMergeAction = "NOOP" | "DELETE" | "REBASE";
