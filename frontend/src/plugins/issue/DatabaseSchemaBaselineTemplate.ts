import {
  IssueCreate,
  PipelineApporvalPolicyPayload,
  StageCreate,
  UNKNOWN_ID,
} from "../../types";
import { IssueTemplate, TemplateContext } from "../types";

const template: IssueTemplate = {
  type: "bb.issue.database.schema.baseline",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueCreate, "projectID" | "creatorID"> => {
    const payload: any = {};
    const stageList: StageCreate[] = [];
    for (let i = 0; i < ctx.databaseList.length; i++) {
      stageList.push({
        name: `[${ctx.environmentList[i].name}] ${ctx.databaseList[i].name}`,
        environmentID: ctx.environmentList[i].id,
        taskList: [
          {
            name: `Establish ${ctx.databaseList[i].name} baseline`,
            status:
              (
                ctx.approvalPolicyList[i]
                  .payload as PipelineApporvalPolicyPayload
              ).value == "MANUAL_APPROVAL_ALWAYS"
                ? "PENDING_APPROVAL"
                : "PENDING",
            type: "bb.task.database.schema.update",
            instanceID: ctx.databaseList[i].instance.id,
            databaseID: ctx.databaseList[i].id,
            statement: "/* Establish baseline using current schema */",
            rollbackStatement: "",
            migrationType: "BASELINE",
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
      assigneeID: UNKNOWN_ID,
      pipeline: {
        stageList,
        name:
          ctx.databaseList.length == 1
            ? `[${ctx.databaseList[0].name}] Establish baseline pipeline`
            : "Establish database baseline pipeline",
      },
      payload,
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
