import { TaskTemplate, TemplateContext, TaskBuiltinFieldId } from "../types";

import { TaskNew, DatabaseId, UNKNOWN_ID } from "../../types";

const template: TaskTemplate = {
  type: "bytebase.database.schema.update",
  buildTask: (ctx: TemplateContext): TaskNew => {
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
      payload: {},
    };
  },
  fieldList: [
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.DATABASE,
      slug: "db",
      name: "DB Name",
      type: "Database",
      required: true,
      isEmpty: (value: DatabaseId): boolean => {
        return value != undefined && value != UNKNOWN_ID;
      },
    },
  ],
};

export default template;
