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
    return {
      name: "Update database schema",
      type: "bytebase.database.schema.update",
      description: "",
      pipeline: {
        taskList: ctx.databaseList.map(
          (database: Database): TaskNew => {
            return {
              name: `[${database.instance.environment.name}] ${database.name}`,
              type: "bytebase.task.schema.update",
              databaseId: database.id,
              stepList: [
                {
                  name: "Waiting for approval",
                  type: "bytebase.step.approve",
                },
                {
                  name: `Update ${database.name} schema`,
                  type: "bytebase.step.database.schema.update",
                },
              ],
            };
          }
        ),
        creatorId: ctx.currentUser.id,
        name: "Update database schema pipeline",
      },
      payload,
    };
  },
  fieldList: [],
};

export default template;
