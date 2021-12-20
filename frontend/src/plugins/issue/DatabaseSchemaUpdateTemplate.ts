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
    const updateSchemaDetails = [];
    for (let i = 0; i < ctx.databaseList.length; i++) {
      updateSchemaDetails.push({
        instanceId: ctx.databaseList[i].instance.id,
        databaseId: ctx.databaseList[i].id,
        statement: ctx.statementList ? ctx.statementList[i] : "",
        rollbackStatement: ctx.rollbackStatementList ? ctx.rollbackStatementList[i] : "",
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
        stageList: [],
        name: "",
      },
      createContext: {
        migrationType: "MIGRATE",
        updateSchemaDetailList: updateSchemaDetails,
      },
      payload,
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
