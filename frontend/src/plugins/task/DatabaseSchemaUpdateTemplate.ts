import { TaskTemplate, TemplateContext, TaskBuiltinFieldId } from "../types";

import { TaskNew } from "../../types";

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
    },
  ],
};

export default template;
