import isEmpty from "lodash-es/isEmpty";

import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  IssueContext,
} from "../types";

import {
  Database,
  Stage,
  StageCreate,
  IssueCreate,
  UNKNOWN_ID,
} from "../../types";

const template: IssueTemplate = {
  type: "bb.issue.database.schema.update",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueCreate, "projectId" | "creatorId"> => {
    const payload: any = {};
    const stageList: StageCreate[] = [];
    for (let i = 0; i < ctx.databaseList.length; i++) {
      stageList.push({
        name: `[${ctx.databaseList[i].instance.environment.name}] ${ctx.databaseList[i].name}`,
        type: "bb.stage.schema.update",
        environmentId: ctx.environmentList[i].id,
        taskList: [
          {
            name: `Update ${ctx.databaseList[i].name} schema`,
            status: "PENDING_APPROVAL",
            type: "bb.task.database.schema.update",
            databaseId: ctx.databaseList[i].id,
          },
        ],
      });
    }
    return {
      name: "Update database schema",
      type: "bb.issue.database.schema.update",
      description: "",
      pipeline: {
        stageList,
        creatorId: ctx.currentUser.id,
        name: "Update database schema pipeline",
      },
      payload,
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
