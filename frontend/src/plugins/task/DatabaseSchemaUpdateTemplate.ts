import isEmpty from "lodash-es/isEmpty";

import { TaskTemplate, TemplateContext, TaskBuiltinFieldId } from "../types";

import { TaskNew, DatabaseId, EnvironmentId, UNKNOWN_ID } from "../../types";

const template: TaskTemplate = {
  type: "bytebase.database.schema.update",
  buildTask: (ctx: TemplateContext): TaskNew => {
    const payload: any = {};
    if (ctx.environmentList.length > 0) {
      // Set the last element as the default value.
      // Normally the last environment is the prod env and is most commonly used.
      payload[TaskBuiltinFieldId.ENVIRONMENT] =
        ctx.environmentList[ctx.environmentList.length - 1].id;
    }
    return {
      name: "Update Schema",
      type: "bytebase.database.schema.update",
      description: "DDL: ",
      stageProgressList: ctx.environmentList.map((env) => {
        return {
          id: env.id,
          name: env.name,
          type: "ENVIRONMENT",
          status: "PENDING",
          runnable: {
            auto: true,
            run: () => {
              console.log("Start", env.name);
            },
          },
        };
      }),
      creatorId: ctx.currentUser.id,
      payload,
    };
  },
  fieldList: [
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.ENVIRONMENT,
      slug: "env",
      name: "Environment",
      type: "Environment",
      required: true,
      isEmpty: (value: EnvironmentId): boolean => {
        return isEmpty(value);
      },
    },
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.DATABASE,
      slug: "db",
      name: "Database",
      type: "Database",
      required: true,
      isEmpty: (value: DatabaseId): boolean => {
        return value == undefined || value == UNKNOWN_ID;
      },
    },
  ],
};

export default template;
