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
  ): Omit<IssueCreate, "projectId" | "creatorId"> => {
    const payload: any = {};
    const stageList: StageCreate[] = [];
    for (let i = 0; i < ctx.databaseList.length; i++) {
      stageList.push({
        name: `[${ctx.environmentList[i].name}] ${ctx.databaseList[i].name}`,
        environmentId: ctx.environmentList[i].id,
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
            instanceId: ctx.databaseList[i].instance.id,
            databaseId: ctx.databaseList[i].id,
            statement: ctx.statementList ? ctx.statementList[i] : "",
            rollbackStatement: "",
            migrationType: "MIGRATE",
            earliestAllowedTs: 0,
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
      assigneeId: UNKNOWN_ID,
      pipeline: {
        stageList,
        name:
          ctx.databaseList.length == 1
            ? `[${ctx.databaseList[0].name}] Update schema pipeline`
            : "Update database schema pipeline",
      },
      createContext: {},
      payload,
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
