import { IssueTemplate, TemplateContext } from "../types";
import { IssueCreate } from "../../types";

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
