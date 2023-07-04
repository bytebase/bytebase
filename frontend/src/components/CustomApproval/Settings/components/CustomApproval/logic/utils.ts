import { v4 as uuidv4 } from "uuid";
import { useCurrentUserV1 } from "@/store";
import { LocalApprovalRule } from "@/types";
import { ApprovalTemplate } from "@/types/proto/v1/issue_service";
import { userNamePrefix } from "@/store/modules/v1/common";

export const emptyLocalApprovalRule = (): LocalApprovalRule => {
  return {
    uid: uuidv4(),
    template: ApprovalTemplate.fromJSON({
      creator: `${userNamePrefix}${useCurrentUserV1().value.email}`,
      flow: {
        steps: [],
      },
    }),
  };
};
