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
  ): Omit<IssueCreate, "projectId" | "creatorId"> => {
    const payload: any = {};
    const updateSchemaDetails = [];
    for (let i = 0; i < ctx.databaseList.length; i++) {
      updateSchemaDetails.push({
        instanceId: ctx.databaseList[i].instance.id,
        databaseId: ctx.databaseList[i].id,
        statement: "/* Establish baseline using current schema */",
        rollbackStatement: "",
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
        stageList: [],
        name: "",
      },
      createContext: {
        migrationType: "BASELINE",
        updateSchemaDetailList: updateSchemaDetails,
      },
      payload,
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
