import isEmpty from "lodash-es/isEmpty";
import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  INPUT_CUSTOM_FIELD_ID_BEGIN,
  IssueContext,
  OUTPUT_CUSTOM_FIELD_ID_BEGIN,
} from "../types";
import { IssueCreate, EnvironmentId, UNKNOWN_ID, Issue } from "../../types";
import { allowDatabaseAccess, fullDatabasePath } from "../../utils";

const INPUT_READ_ONLY_FIELD_ID = INPUT_CUSTOM_FIELD_ID_BEGIN;
const OUTPUT_DATABASE_FIELD_ID = OUTPUT_CUSTOM_FIELD_ID_BEGIN;

const template: IssueTemplate = {
  type: "bb.issue.database.grant",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueCreate, "projectId" | "creatorId"> => {
    const payload: any = {};

    return {
      name: "Request database access",
      type: "bb.issue.database.grant",
      description: "",
      pipeline: {
        stageList: [
          {
            name: "Request database access",
            environmentId: ctx.environmentList[0].id,
            taskList: [
              {
                name: "Request database access",
                status: "PENDING_APPROVAL",
                type: "bb.task.general",
                instanceId: ctx.databaseList[0].instance.id,
                databaseId: ctx.databaseList[0].id,
                statement: "",
                rollbackStatement: "",
              },
            ],
          },
        ],
        name: "Request database access",
      },
      payload,
    };
  },
  inputFieldList: [
    {
      id: INPUT_READ_ONLY_FIELD_ID,
      slug: "readonly",
      name: "Read Only",
      type: "Boolean",
      allowEditAfterCreation: false,
      resolved: (ctx: IssueContext): boolean => {
        return true;
      },
    },
  ],
  outputFieldList: [
    {
      id: OUTPUT_DATABASE_FIELD_ID,
      name: "Granted database",
      type: "Database",
      resolved: (ctx: IssueContext): boolean => {
        const issue = ctx.issue as Issue;
        const database = issue.pipeline.stageList[0].taskList[0].database!;
        const creator = (ctx.issue as Issue).creator;
        const type = ctx.issue.payload[INPUT_READ_ONLY_FIELD_ID] ? "RO" : "RW";
        return allowDatabaseAccess(database, creator, type);
      },
      actionText: "+ Grant",
      actionLink: (ctx: IssueContext): string => {
        const queryParamList: string[] = [];

        const issue = ctx.issue as Issue;
        const database = issue.pipeline.stageList[0].taskList[0].database!;
        const readOnly = issue.payload[INPUT_READ_ONLY_FIELD_ID];
        let dataSourceId;
        for (const dataSource of database.dataSourceList) {
          if (readOnly && dataSource.type == "RO") {
            dataSourceId = dataSource.id;
            break;
          } else if (!readOnly && dataSource.type == "RW") {
            dataSourceId = dataSource.id;
            break;
          }
        }

        if (dataSourceId) {
          queryParamList.push(`database=${database.id}`);

          queryParamList.push(`datasource=${dataSourceId}`);

          queryParamList.push(`grantee=${issue.creator.id}`);

          queryParamList.push(`issue=${issue.id}`);

          return "/db/grant?" + queryParamList.join("&");
        }
        return "";
      },
      viewLink: (ctx: IssueContext): string => {
        const issue = ctx.issue as Issue;
        const database = issue.pipeline.stageList[0].taskList[0].database!;
        if (database.id != UNKNOWN_ID) {
          return fullDatabasePath(database);
        }
        return "";
      },
      resolveStatusText: (resolved: boolean): string => {
        return resolved ? "(Granted)" : "(To be granted)";
      },
    },
  ],
};

export default template;
