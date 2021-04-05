import isEmpty from "lodash-es/isEmpty";
import { TaskNew, UNKNOWN_ID } from "../../types";
import {
  TaskBuiltinFieldId,
  TaskContext,
  TaskTemplate,
  TemplateContext,
} from "../types";

const template: TaskTemplate = {
  type: "bytebase.general",
  buildTask: (ctx: TemplateContext): TaskNew => {
    return {
      name: "",
      type: "bytebase.general",
      description: "",
      stageList: [
        {
          id: "1",
          name: "Request",
          type: "bytebase.stage.general",
          status: "PENDING",
        },
      ],
      creatorId: ctx.currentUser.id,
      subscriberIdList: [],
      payload: {},
    };
  },
  fieldList: [
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.ENVIRONMENT,
      slug: "environment",
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
      slug: "database",
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
