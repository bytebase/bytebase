import { Status } from "nice-grpc-common";
import { Branch } from "@/types/proto/v1/branch_service";

export type MergeBranchValidationState = {
  status: Status;
  branch?: Branch;
};
