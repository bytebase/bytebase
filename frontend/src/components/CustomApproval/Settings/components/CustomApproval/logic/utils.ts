import { v4 as uuidv4 } from "uuid";
import { useCurrentUser } from "@/store";
import { LocalApprovalRule } from "@/types";
import { ApprovalTemplate } from "@/types/proto/store/approval";

export const emptyLocalApprovalRule = (): LocalApprovalRule => {
  return {
    uid: uuidv4(),
    template: ApprovalTemplate.fromJSON({
      creatorId: useCurrentUser().value.id,
      flow: {
        steps: [],
      },
    }),
  };
};
