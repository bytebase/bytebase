import { Issue, IssueCreate, UNKNOWN_ID } from "../../types";
import { allowDatabaseAccess, fullDatabasePath } from "../../utils";
import {
  INPUT_CUSTOM_FIELD_ID_BEGIN,
  IssueContext,
  IssueTemplate,
  OUTPUT_CUSTOM_FIELD_ID_BEGIN,
  TemplateContext,
} from "../types";

const INPUT_READ_ONLY_FIELD_ID = INPUT_CUSTOM_FIELD_ID_BEGIN;
const OUTPUT_DATABASE_FIELD_ID = OUTPUT_CUSTOM_FIELD_ID_BEGIN;

const template: IssueTemplate = {
  type: "bb.issue.database.grant",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueCreate, "projectID" | "creatorID"> => {
    const payload: any = {};

    return {
      name: "Request database access",
      type: "bb.issue.database.grant",
      description: "",
      assigneeID: UNKNOWN_ID,
      pipeline: {
        stageList: [
          {
            name: "Request database access",
            environmentID: ctx.environmentList[0].id,
            taskList: [
              {
                name: "Request database access",
                status: "PENDING_APPROVAL",
                type: "bb.task.general",
                instanceID: ctx.databaseList[0].instance.id,
                databaseID: ctx.databaseList[0].id,
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
        const readonly = issue.payload[INPUT_READ_ONLY_FIELD_ID];
        let dataSourceID;
        for (const dataSource of database.dataSourceList) {
          if (readonly && dataSource.type == "RO") {
            dataSourceID = dataSource.id;
            break;
          } else if (!readonly && dataSource.type == "RW") {
            dataSourceID = dataSource.id;
            break;
          }
        }

        if (dataSourceID) {
          queryParamList.push(`database=${database.id}`);

          queryParamList.push(`datasource=${dataSourceID}`);

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
