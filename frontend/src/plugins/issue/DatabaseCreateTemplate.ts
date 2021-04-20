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
        taskList: [
          {
            name: "Create database",
            environmentId: ctx.environmentList[0].id,
            databaseId: EMPTY_ID,
            type: "bytebase.task.database.create",
            stepList: [
              {
                name: "Waiting for approval",
                type: "bytebase.step.approve",
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
  fieldList: [
    {
      category: "INPUT",
      id: INPUT_DATABASE_NAME,
      slug: "databaseName",
      name: "DB name",
      type: "NewDatabase",
      required: true,
      resolved: (ctx: IssueContext): boolean => {
        const databaseName = ctx.issue.payload[INPUT_DATABASE_NAME];
        return !isEmpty(databaseName);
      },
      placeholder: "New database name",
    },
    {
      category: "OUTPUT",
      id: OUTPUT_DATABASE_FIELD_ID,
      slug: "",
      name: "Created database",
      type: "Database",
      required: true,
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
