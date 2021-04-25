import isEmpty from "lodash-es/isEmpty";
import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  OUTPUT_CUSTOM_FIELD_ID_BEGIN,
  IssueContext,
  INPUT_CUSTOM_FIELD_ID_BEGIN,
} from "../types";
import { EMPTY_ID, Issue, IssueNew, Pipeline, UNKNOWN_ID } from "../../types";
import { activeEnvironment, fullDatabasePath } from "../../utils";

const INPUT_DATABASE_NAME = INPUT_CUSTOM_FIELD_ID_BEGIN;
const OUTPUT_DATABASE_FIELD_ID = OUTPUT_CUSTOM_FIELD_ID_BEGIN;

const template: IssueTemplate = {
  type: "bb.database.create",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueNew, "projectId" | "creatorId"> => {
    const payload: any = {};

    return {
      name: "Request new db",
      type: "bb.database.create",
      description: "",
      pipeline: {
        stageList: [
          {
            name: "Create database",
            environmentId: ctx.environmentList[0].id,
            type: "bb.stage.database.create",
            taskList: [
              {
                name: "Waiting for approval",
                type: "bb.task.approve",
                when: "MANUAL",
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
      type: "String",
      // Developer might create a database name which already exists, so we give the assignee the ability to change.
      allowEditAfterCreation: true,
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
        return database.name == requestedName;
      },
      actionText: "+ Create",
      actionLink: (ctx: IssueContext): string => {
        const queryParamList: string[] = [];

        const issue = ctx.issue as Issue;

        queryParamList.push(`project=${issue.project.id}`);

        const environment = activeEnvironment(issue.pipeline!);
        queryParamList.push(`environment=${environment.id}`);

        const databaseName = issue.payload[INPUT_DATABASE_NAME];
        queryParamList.push(`name=${databaseName}`);

        queryParamList.push(`issue=${issue.id}`);

        queryParamList.push(`from=${issue.type}`);

        return "/db/new?" + queryParamList.join("&");
      },
      viewLink: (ctx: IssueContext): string => {
        const databaseId = ctx.issue.payload[OUTPUT_DATABASE_FIELD_ID];
        const database = ctx.store.getters["database/databaseById"](databaseId);
        if (database.id != UNKNOWN_ID) {
          return fullDatabasePath(database);
        }
        return "";
      },
      resolveStatusText: (resolved: boolean): string => {
        return resolved ? "(Created)" : "(To be created)";
      },
    },
  ],
};

export default template;
