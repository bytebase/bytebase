import {
  IssueCreate,
  PipelineApporvalPolicyPayload,
  StageCreate,
  UNKNOWN_ID,
} from "../../types";
import { IssueTemplate, TemplateContext } from "../types";

const template: IssueTemplate = {
  type: "bb.issue.database.schema.update",
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
            name: `Update ${ctx.databaseList[i].name} schema`,
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
            statement: ctx.statementList ? ctx.statementList[i] : "",
            rollbackStatement: ctx.rollbackStatementList
              ? ctx.rollbackStatementList[i]
              : "",
            migrationType: "MIGRATE",
          },
        ],
      });
    }
    return {
      name:
        ctx.databaseList.length == 1
          ? `[${ctx.databaseList[0].name}] Update schema`
          : "Update database schema",
      type: "bb.issue.database.schema.update",
      description: "",
      assigneeID: UNKNOWN_ID,
      pipeline: {
        stageList,
        name:
          ctx.databaseList.length == 1
            ? `[${ctx.databaseList[0].name}] Update schema pipeline`
            : "Update database schema pipeline",
      },
      payload,
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
