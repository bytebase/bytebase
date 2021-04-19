import isEmpty from "lodash-es/isEmpty";

import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  IssueContext,
} from "../types";

import { Database, Stage, StageNew, IssueNew, UNKNOWN_ID } from "../../types";

const template: IssueTemplate = {
  type: "bytebase.database.schema.update",
  buildIssue: (ctx: TemplateContext): IssueNew => {
    const payload: any = {};
    return {
      name: "Change db",
      type: "bytebase.database.schema.update",
      description: "",
      stageList: ctx.databaseList.map(
        (database: Database): StageNew => {
          return {
            name: `[${database.instance.environment.name}] ${database.name}`,
            type: "bytebase.stage.schema.update",
            databaseId: database.id,
            stepList: [
              {
                name: "Waiting for approval",
                type: "bytebase.step.approve",
              },
              {
                name: `Update ${database.name} schema`,
                type: "bytebase.step.database.schema.update",
              },
            ],
          };
        }
      ),
      creatorId: ctx.currentUser.id,
      subscriberIdList: [],
      payload,
    };
  },
  fieldList: [],
};

export default template;
