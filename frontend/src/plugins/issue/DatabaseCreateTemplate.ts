import { IssueCreate, UNKNOWN_ID } from "../../types";
import { IssueTemplate, TemplateContext } from "../types";

const template: IssueTemplate = {
  type: "bb.issue.database.create",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueCreate, "projectID" | "creatorID"> => {
    const payload: any = {};

    return {
      name: "Create database",
      type: "bb.issue.database.create",
      description: "",
      assigneeID: UNKNOWN_ID,
      pipeline: {
        stageList: [
          {
            name: "Create database",
            environmentID: ctx.environmentList[0].id,
            taskList: [
              {
                name: "Create database",
                status: "PENDING_APPROVAL",
                type: "bb.task.database.create",
                instanceID: ctx.databaseList[0].instance.id,
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
