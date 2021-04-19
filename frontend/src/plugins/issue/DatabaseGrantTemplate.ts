import isEmpty from "lodash-es/isEmpty";
import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  INPUT_CUSTOM_FIELD_ID_BEGIN,
  IssueContext,
} from "../types";
import { IssueNew, EnvironmentId, UNKNOWN_ID, Issue } from "../../types";
import { allowDatabaseAccess } from "../../utils";

const INPUT_READ_ONLY_FIELD_ID = INPUT_CUSTOM_FIELD_ID_BEGIN;

const template: IssueTemplate = {
  type: "bytebase.database.grant",
  buildIssue: (ctx: TemplateContext): IssueNew => {
    const payload: any = {};

    return {
      name: "Request database access",
      type: "bytebase.database.grant",
      description: "",
      stageList: [
        {
          name: "Request database access",
          type: "bytebase.stage.database.grant",
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
      name: "Database",
      type: "Database",
      required: true,
      resolved: (ctx: IssueContext): boolean => {
        const databaseId = ctx.issue.payload[IssueBuiltinFieldId.DATABASE];
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
      resolved: (ctx: IssueContext): boolean => {
        return true;
      },
    },
    {
      category: "OUTPUT",
      // This is the same ID as the INPUT database field because the granted database should be the same
      // as the requested database.
      id: IssueBuiltinFieldId.DATABASE,
      slug: "database",
      name: "Granted database",
      type: "Database",
      required: true,
      resolved: (ctx: IssueContext): boolean => {
        const databaseId = ctx.issue.payload[IssueBuiltinFieldId.DATABASE];
        const database = ctx.store.getters["database/databaseById"](databaseId);
        const creator = (ctx.issue as Issue).creator;
        const type = ctx.issue.payload[INPUT_READ_ONLY_FIELD_ID] ? "RO" : "RW";
        return allowDatabaseAccess(database, creator, type);
      },
    },
  ],
};

export default template;
