import isEmpty from "lodash-es/isEmpty";
import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  OUTPUT_CUSTOM_FIELD_ID_BEGIN,
  IssueContext,
  INPUT_CUSTOM_FIELD_ID_BEGIN,
} from "../types";
import { EMPTY_ID, IssueNew, UNKNOWN_ID } from "../../types";

const INPUT_DATABASE_NAME = INPUT_CUSTOM_FIELD_ID_BEGIN;
const OUTPUT_DATABASE_FIELD_ID = OUTPUT_CUSTOM_FIELD_ID_BEGIN;

const template: IssueTemplate = {
  type: "bytebase.database.create",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueNew, "projectId" | "creatorId"> => {
    const payload: any = {};
    payload[IssueBuiltinFieldId.DATABASE] = "";

    return {
      name: "Request new db",
      type: "bytebase.database.create",
      description: "",
      pipeline: {
        stageList: [
          {
            name: "Create database",
            environmentId: ctx.environmentList[0].id,
            type: "bytebase.stage.database.create",
            taskList: [
              {
                name: "Waiting for approval",
                type: "bytebase.task.approve",
              },
            ],
          },
        ],
        creatorId: ctx.currentUser.id,
        name: "Create database pipeline",
      },
      payload,
    };
  },
  inputFieldList: [
    {
      id: INPUT_DATABASE_NAME,
      slug: "databaseName",
      name: "DB name",
      type: "NewDatabase",
      resolved: (ctx: IssueContext): boolean => {
        const databaseName = ctx.issue.payload[INPUT_DATABASE_NAME];
        return !isEmpty(databaseName);
      },
      placeholder: "New database name",
    },
  ],
  outputFieldList: [
    {
      id: OUTPUT_DATABASE_FIELD_ID,
      name: "Created database",
      type: "Database",
      // Returns true if it's set and matches the requested database name.
      resolved: (ctx: IssueContext): boolean => {
        const databaseId = ctx.issue.payload[OUTPUT_DATABASE_FIELD_ID];
        if (isEmpty(databaseId) || databaseId == UNKNOWN_ID) {
          return false;
        }
        const requestedName = ctx.issue.payload[INPUT_DATABASE_NAME];
        const database = ctx.store.getters["database/databaseById"](databaseId);
        return database && database.name == requestedName;
      },
    },
  ],
};

export default template;
