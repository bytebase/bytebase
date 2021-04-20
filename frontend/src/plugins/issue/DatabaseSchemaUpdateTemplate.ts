import isEmpty from "lodash-es/isEmpty";

import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  IssueContext,
} from "../types";

import { Database, Task, TaskNew, IssueNew, UNKNOWN_ID } from "../../types";

const template: IssueTemplate = {
  type: "bytebase.database.schema.update",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueNew, "projectId" | "creatorId"> => {
    const payload: any = {};
    const taskList: TaskNew[] = [];
    for (let i = 0; i < ctx.databaseList.length; i++) {
      taskList.push({
        name: `[${ctx.databaseList[i].instance.environment.name}] ${ctx.databaseList[i].name}`,
        type: "bytebase.task.schema.update",
        environmentId: ctx.environmentList[i].id,
        databaseId: ctx.databaseList[i].id,
        stepList: [
          {
            name: "Waiting for approval",
            type: "bytebase.step.approve",
          },
          {
            name: `Update ${ctx.databaseList[i].name} schema`,
            type: "bytebase.step.database.schema.update",
          },
        ],
      });
    }
    return {
      name: "Update database schema",
      type: "bytebase.database.schema.update",
      description: "",
      pipeline: {
        taskList,
        creatorId: ctx.currentUser.id,
        name: "Update database schema pipeline",
      },
      payload,
    };
  },
  fieldList: [],
};

export default template;
