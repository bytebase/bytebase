import isEmpty from "lodash-es/isEmpty";

import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  IssueContext,
} from "../types";

import { Database, Stage, StageNew, IssueNew, UNKNOWN_ID } from "../../types";

const template: IssueTemplate = {
  type: "bb.database.schema.update",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueNew, "projectId" | "creatorId"> => {
    const payload: any = {};
    const stageList: StageNew[] = [];
    for (let i = 0; i < ctx.databaseList.length; i++) {
      stageList.push({
        name: `[${ctx.databaseList[i].instance.environment.name}] ${ctx.databaseList[i].name}`,
        type: "bb.stage.schema.update",
        environmentId: ctx.environmentList[i].id,
        taskList: [
          {
            name: `Update ${ctx.databaseList[i].name} schema`,
            type: "bb.task.database.schema.update",
            when: "MANUAL",
            databaseId: ctx.databaseList[i].id,
          },
        ],
      });
    }
    return {
      name: "Update database schema",
      type: "bb.database.schema.update",
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
