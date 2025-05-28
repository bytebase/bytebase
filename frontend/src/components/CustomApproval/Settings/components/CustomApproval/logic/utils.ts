import { v4 as uuidv4 } from "uuid";
import type { LocalApprovalRule } from "@/types";
import { ApprovalTemplate } from "@/types/proto/v1/issue_service";

export const emptyLocalApprovalRule = (): LocalApprovalRule => {
  return {
    uid: uuidv4(),
    template: ApprovalTemplate.fromPartial({
      flow: {
        steps: [],
      },
    }),
  };
};
