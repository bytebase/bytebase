import isEmpty from "lodash-es/isEmpty";
import {
  TaskField,
  TaskTemplate,
  TemplateContext,
  TaskBuiltinFieldId,
  DatabaseFieldPayload,
  CUSTOM_FIELD_ID_BEGIN,
} from "../types";
import { linkfy } from "../../utils";
import { Task, TaskNew, EnvironmentId, UNKNOWN_ID } from "../../types";

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
      name: "DB name",
      type: "NewDatabase",
      required: true,
      isEmpty: (dbName: string): boolean => {
        return isEmpty(dbName);
      },
      placeholder: "New database name",
    },
    {
      category: "OUTPUT",
      id: CUSTOM_FIELD_ID_BEGIN + 1,
      slug: "database",
      name: "Created database",
      type: "Database",
      required: true,
      isEmpty: (databaseId: string): boolean => {
        return isEmpty(databaseId) || databaseId == UNKNOWN_ID;
      },
    },
  ],
};

export default template;
