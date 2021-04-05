import isEmpty from "lodash-es/isEmpty";

import {
  TaskTemplate,
  TemplateContext,
  TaskBuiltinFieldId,
  TaskContext,
} from "../types";

import { TaskNew, UNKNOWN_ID } from "../../types";

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
      name: "Change db",
      type: "bytebase.database.schema.update",
      description: "",
      stageList: [
        {
          id: "1",
          name: "Update schema",
          type: "bytebase.stage.schema.update",
          status: "PENDING",
        },
      ],
      creatorId: ctx.currentUser.id,
      subscriberIdList: [],
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
      resolved: (ctx: TaskContext): boolean => {
        const environmentId = ctx.task.payload[TaskBuiltinFieldId.ENVIRONMENT];
        return !isEmpty(environmentId);
      },
    },
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.DATABASE,
      slug: "db",
      name: "Database",
      type: "Database",
      required: true,
      resolved: (ctx: TaskContext): boolean => {
        const databaseId = ctx.task.payload[TaskBuiltinFieldId.DATABASE];
        return !isEmpty(databaseId) || databaseId == UNKNOWN_ID;
      },
    },
  ],
};

export default template;
