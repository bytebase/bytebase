import isEmpty from "lodash-es/isEmpty";
import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  INPUT_CUSTOM_FIELD_ID_BEGIN,
  IssueContext,
  OUTPUT_CUSTOM_FIELD_ID_BEGIN,
} from "../types";
import { IssueNew, EnvironmentId, UNKNOWN_ID, Issue } from "../../types";
import { allowDatabaseAccess } from "../../utils";

const INPUT_READ_ONLY_FIELD_ID = INPUT_CUSTOM_FIELD_ID_BEGIN;
const OUTPUT_DATABASE_FIELD_ID = OUTPUT_CUSTOM_FIELD_ID_BEGIN;

const template: IssueTemplate = {
  type: "bytebase.database.grant",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueNew, "projectId" | "creatorId"> => {
    const payload: any = {};

    return {
      name: "Request database access",
      type: "bytebase.database.grant",
      description: "",
      pipeline: {
        stageList: [
          {
            name: "Request database access",
            type: "bytebase.stage.database.grant",
            environmentId: ctx.environmentList[0].id,
            databaseId: ctx.databaseList[0].id,
            taskList: [
              {
                name: "Waiting for approval",
                type: "bytebase.task.approve",
              },
            ],
          },
        ],
        creatorId: ctx.currentUser.id,
        name: "Request database access",
      },
      payload,
    };
  },
  fieldList: [
    {
      category: "INPUT",
      id: INPUT_READ_ONLY_FIELD_ID,
      slug: "readonly",
      name: "Read Only",
      type: "Boolean",
      required: true,
      resolved: (ctx: IssueContext): boolean => {
        return true;
      },
    },
    {
      category: "OUTPUT",
      // This is the same ID as the INPUT database field because the granted database should be the same
      // as the requested database.
      id: OUTPUT_DATABASE_FIELD_ID,
      slug: "",
      name: "Granted database",
      type: "Database",
      required: true,
      resolved: (ctx: IssueContext): boolean => {
        const databaseId = ctx.issue.payload[OUTPUT_DATABASE_FIELD_ID];
        const database = ctx.store.getters["database/databaseById"](databaseId);
        const creator = (ctx.issue as Issue).creator;
        const type = ctx.issue.payload[INPUT_READ_ONLY_FIELD_ID] ? "RO" : "RW";
        return allowDatabaseAccess(database, creator, type);
      },
    },
  ],
};

export default template;
