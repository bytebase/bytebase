import { defaultApprovalStrategy } from "@/store/modules/v1/policy";
import { ApprovalStrategy } from "@/types/proto/v1/org_policy_service";
import { IssueCreate, StageCreate, UNKNOWN_ID } from "../../types";
import { IssueTemplate, TemplateContext } from "../types";

const template: IssueTemplate = {
  type: "bb.issue.database.schema.baseline",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueCreate, "projectId" | "creatorId"> => {
    const payload: any = {};
    const stageList: StageCreate[] = [];
    for (let i = 0; i < ctx.databaseList.length; i++) {
      stageList.push({
        name: `[${ctx.environmentList[i].name}] ${ctx.databaseList[i].name}`,
        environmentId: ctx.environmentList[i].id,
        taskList: [
          {
            name: `Establish ${ctx.databaseList[i].name} baseline`,
            status:
              (ctx.approvalPolicyList[i].deploymentApprovalPolicy
                ?.defaultStrategy ?? defaultApprovalStrategy) ==
              ApprovalStrategy.MANUAL
                ? "PENDING_APPROVAL"
                : "PENDING",
            type: "bb.task.database.schema.update",
            instanceId: ctx.databaseList[i].instance.id,
            databaseId: ctx.databaseList[i].id,
            statement: "/* Establish baseline using current schema */",
            sheetId: UNKNOWN_ID,
            earliestAllowedTs: 0,
          },
        ],
      });
    }
    return {
      name:
        ctx.databaseList.length == 1
          ? `[${ctx.databaseList[0].name}] Establish baseline`
          : "Establish database baseline",
      type: "bb.issue.database.schema.update",
      description: "",
      assigneeId: UNKNOWN_ID,
      pipeline: {
        stageList,
        name:
          ctx.databaseList.length == 1
            ? `[${ctx.databaseList[0].name}] Establish baseline pipeline`
            : "Establish database baseline pipeline",
      },
      createContext: {},
      payload,
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
