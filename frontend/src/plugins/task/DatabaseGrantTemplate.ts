import isEmpty from "lodash-es/isEmpty";
import {
  TaskTemplate,
  TemplateContext,
  TaskBuiltinFieldId,
  INPUT_CUSTOM_FIELD_ID_BEGIN,
  TaskContext,
} from "../types";
import { TaskNew, EnvironmentId, UNKNOWN_ID, Task } from "../../types";
import { allowDatabaseAccess } from "../../utils";

const INPUT_READ_ONLY_FIELD_ID = INPUT_CUSTOM_FIELD_ID_BEGIN;

const template: TaskTemplate = {
  type: "bytebase.database.grant",
  buildTask: (ctx: TemplateContext): TaskNew => {
    const payload: any = {};

    return {
      name: "Request database access",
      type: "bytebase.database.grant",
      description: "",
      stageList: [
        {
          id: "1",
          name: "Request database access",
          type: "bytebase.stage.database.grant",
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
    {
      category: "INPUT",
      id: INPUT_READ_ONLY_FIELD_ID,
      slug: "readonly",
      name: "Read Only",
      type: "Boolean",
      required: true,
      resolved: (ctx: TaskContext): boolean => {
        return true;
      },
    },
    {
      category: "OUTPUT",
      // This is the same ID as the INPUT database field because the granted database should be the same
      // as the requested database.
      id: TaskBuiltinFieldId.DATABASE,
      slug: "database",
      name: "Granted database",
      type: "Database",
      required: true,
      resolved: (ctx: TaskContext): boolean => {
        const databaseId = ctx.task.payload[TaskBuiltinFieldId.DATABASE];
        const database = ctx.store.getters["database/databaseById"](databaseId);
        const creator = (ctx.task as Task).creator;
        const type = ctx.task.payload[INPUT_READ_ONLY_FIELD_ID] ? "RO" : "RW";
        return allowDatabaseAccess(database, creator, type);
      },
    },
  ],
};

export default template;
