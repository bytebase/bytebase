import { create as createProto } from "@bufbuild/protobuf";
import { v4 as uuidv4 } from "uuid";
import type { LocalApprovalRule } from "@/types";
import { ApprovalTemplateSchema } from "@/types/proto-es/v1/issue_service_pb";

export const emptyLocalApprovalRule = (): LocalApprovalRule => {
  return {
    template: createProto(ApprovalTemplateSchema, {
      id: uuidv4(),
      flow: {
        roles: [],
      },
    }),
  };
};
