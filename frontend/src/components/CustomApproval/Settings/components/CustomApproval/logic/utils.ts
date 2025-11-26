import { create as createProto } from "@bufbuild/protobuf";
import { v4 as uuidv4 } from "uuid";
import type { LocalApprovalRule } from "@/types";
import { ApprovalFlowSchema } from "@/types/proto-es/v1/issue_service_pb";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";

export const emptyLocalApprovalRule = (): LocalApprovalRule => {
  return {
    uid: uuidv4(),
    source: WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED,
    title: "",
    description: "",
    condition: "",
    flow: createProto(ApprovalFlowSchema, { roles: [] }),
  };
};
