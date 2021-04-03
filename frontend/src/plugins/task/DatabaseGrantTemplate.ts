import isEmpty from "lodash-es/isEmpty";
import {
  TaskTemplate,
  TemplateContext,
  TaskBuiltinFieldId,
  CUSTOM_FIELD_ID_BEGIN,
} from "../types";
import { TaskNew, EnvironmentId, UNKNOWN_ID } from "../../types";

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
      isEmpty: (value: EnvironmentId): boolean => {
        return isEmpty(value);
      },
    },
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.DATABASE,
      slug: "database",
      name: "Database",
      type: "Database",
      required: true,
      isEmpty: (databaseId: string): boolean => {
        return isEmpty(databaseId) || databaseId == UNKNOWN_ID;
      },
    },
    {
      category: "INPUT",
      id: CUSTOM_FIELD_ID_BEGIN,
      slug: "readonly",
      name: "Read Only",
      type: "Boolean",
      required: true,
      isEmpty: (readOnly: boolean): boolean => {
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
      isEmpty: (databaseId: string): boolean => {
        return isEmpty(databaseId) || databaseId == UNKNOWN_ID;
      },
    },
  ],
};

export default template;
