import isEmpty from "lodash-es/isEmpty";
import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  OUTPUT_CUSTOM_FIELD_ID_BEGIN,
  IssueContext,
} from "../types";
import { IssueNew, UNKNOWN_ID } from "../../types";

const OUTPUT_DATABASE_FIELD_ID = OUTPUT_CUSTOM_FIELD_ID_BEGIN;

const template: IssueTemplate = {
  type: "bytebase.database.create",
  buildIssue: (ctx: TemplateContext): IssueNew => {
    const payload: any = {};
    payload[IssueBuiltinFieldId.DATABASE] = "";

    return {
      name: "Request new db",
      type: "bytebase.database.create",
      description: "",
      stageList: [
        {
          name: "Create database",
          type: "bytebase.stage.database.create",
          databaseId:
            ctx.databaseList.length > 0 ? ctx.databaseList[0].id : UNKNOWN_ID,
          stepList: [
            {
              name: "Waiting for approval",
              type: "bytebase.step.approve",
            },
          ],
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
      id: IssueBuiltinFieldId.PROJECT,
      slug: "project",
      name: "Project",
      type: "Project",
      required: true,
      resolved: (ctx: IssueContext): boolean => {
        const projectId = ctx.issue.payload[IssueBuiltinFieldId.PROJECT];
        return !isEmpty(projectId);
      },
    },
    {
      category: "INPUT",
      id: IssueBuiltinFieldId.ENVIRONMENT,
      slug: "environment",
      name: "Environment",
      type: "Environment",
      required: true,
      resolved: (ctx: IssueContext): boolean => {
        const environmentId =
          ctx.issue.payload[IssueBuiltinFieldId.ENVIRONMENT];
        return !isEmpty(environmentId);
      },
    },
    {
      category: "INPUT",
      id: IssueBuiltinFieldId.DATABASE,
      slug: "database",
      name: "DB name",
      type: "NewDatabase",
      required: true,
      resolved: (ctx: IssueContext): boolean => {
        const databaseName = ctx.issue.payload[IssueBuiltinFieldId.DATABASE];
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
      resolved: (ctx: IssueContext): boolean => {
        const databaseId = ctx.issue.payload[OUTPUT_DATABASE_FIELD_ID];
        if (isEmpty(databaseId) || databaseId == UNKNOWN_ID) {
          return false;
        }
        const requestedName = ctx.issue.payload[IssueBuiltinFieldId.DATABASE];
        const database = ctx.store.getters["database/databaseById"](databaseId);
        return database && database.name == requestedName;
      },
    },
  ],
};

export default template;
