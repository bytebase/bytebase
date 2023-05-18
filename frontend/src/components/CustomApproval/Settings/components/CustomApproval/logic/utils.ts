import { v4 as uuidv4 } from "uuid";
import { useCurrentUserV1 } from "@/store";
import { LocalApprovalRule } from "@/types";
import { ApprovalTemplate } from "@/types/proto/store/approval";
import { extractUserUID } from "@/utils";

export const emptyLocalApprovalRule = (): LocalApprovalRule => {
  return {
    uid: uuidv4(),
    template: ApprovalTemplate.fromJSON({
      creatorId: extractUserUID(useCurrentUserV1().value.name),
      flow: {
        steps: [],
      },
    }),
  };
};
