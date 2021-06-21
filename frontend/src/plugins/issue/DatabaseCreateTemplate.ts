import isEmpty from "lodash-es/isEmpty";
import {
  IssueTemplate,
  TemplateContext,
  IssueBuiltinFieldId,
  OUTPUT_CUSTOM_FIELD_ID_BEGIN,
  IssueContext,
  INPUT_CUSTOM_FIELD_ID_BEGIN,
} from "../types";
import {
  EMPTY_ID,
  Issue,
  IssueCreate,
  Pipeline,
  UNKNOWN_ID,
} from "../../types";
import { activeEnvironment, fullDatabasePath } from "../../utils";

const INPUT_DATABASE_NAME = INPUT_CUSTOM_FIELD_ID_BEGIN;
const OUTPUT_DATABASE_FIELD_ID = OUTPUT_CUSTOM_FIELD_ID_BEGIN;

const template: IssueTemplate = {
  type: "bb.issue.database.create",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueCreate, "projectId" | "creatorId"> => {
    const payload: any = {};

    return {
      name: "Create database",
      type: "bb.issue.database.create",
      description: "",
      pipeline: {
        stageList: [
          {
            name: "Create database",
            environmentId: ctx.environmentList[0].id,
            taskList: [
              {
                name: "Create database",
                status: "PENDING_APPROVAL",
                type: "bb.task.database.create",
                instanceId: ctx.databaseList[0].instance.id,
                statement: "",
                rollbackStatement: "",
              },
            ],
          },
        ],
        name: "Pipeline - Create database",
      },
      payload,
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
