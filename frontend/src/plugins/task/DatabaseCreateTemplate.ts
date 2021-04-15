import isEmpty from "lodash-es/isEmpty";
import {
  TaskTemplate,
  TemplateContext,
  TaskBuiltinFieldId,
  OUTPUT_CUSTOM_FIELD_ID_BEGIN,
  TaskContext,
} from "../types";
import { TaskNew, UNKNOWN_ID } from "../../types";

const OUTPUT_DATABASE_FIELD_ID = OUTPUT_CUSTOM_FIELD_ID_BEGIN;

const template: TaskTemplate = {
  type: "bytebase.database.create",
  buildTask: (ctx: TemplateContext): TaskNew => {
    const payload: any = {};
    if (ctx.environmentList.length > 0) {
      // Set the last element as the default value.
      // Normally the last environment is the prod env and is most commonly used.
      payload[TaskBuiltinFieldId.ENVIRONMENT] =
        ctx.environmentList[ctx.environmentList.length - 1].id;
    }
    payload[TaskBuiltinFieldId.DATABASE] = "";
    return {
      projectId: UNKNOWN_ID,
      name: "Request new db",
      type: "bytebase.database.create",
      description: "",
      stageList: [
        {
          id: "1",
          name: "Create database",
          type: "bytebase.stage.database.create",
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
      id: TaskBuiltinFieldId.PROJECT,
      slug: "project",
      name: "Project",
      type: "Project",
      required: true,
      resolved: (ctx: TaskContext): boolean => {
        const projectId = ctx.task.payload[TaskBuiltinFieldId.PROJECT];
        return !isEmpty(projectId);
      },
    },
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
      name: "DB name",
      type: "NewDatabase",
      required: true,
      resolved: (ctx: TaskContext): boolean => {
        const databaseName = ctx.task.payload[TaskBuiltinFieldId.DATABASE];
        return !isEmpty(databaseName);
      },
      placeholder: "New database name",
    },
    {
      category: "OUTPUT",
      id: OUTPUT_DATABASE_FIELD_ID,
      slug: "database",
      name: "Created database",
      type: "Database",
      required: true,
      // Returns true if it's set and matches the requested database name.
      resolved: (ctx: TaskContext): boolean => {
        const databaseId = ctx.task.payload[OUTPUT_DATABASE_FIELD_ID];
        if (isEmpty(databaseId) || databaseId == UNKNOWN_ID) {
          return false;
        }
        const requestedName = ctx.task.payload[TaskBuiltinFieldId.DATABASE];
        const database = ctx.store.getters["database/databaseById"](databaseId);
        return database && database.name == requestedName;
      },
    },
  ],
};

export default template;
