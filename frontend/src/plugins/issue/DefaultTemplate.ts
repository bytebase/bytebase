import { IssueCreate, UNKNOWN_ID } from "../../types";
import { IssueTemplate, TemplateContext } from "../types";

const template: IssueTemplate = {
  type: "bb.issue.general",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueCreate, "projectID" | "creatorID"> => {
    return {
      name: "",
      type: "bb.issue.general",
      description: "",
      assigneeID: UNKNOWN_ID,
      pipeline: {
        stageList: [
          {
            name: "Troubleshoot database",
            environmentID: UNKNOWN_ID,
            taskList: [
              {
                name: "Troubleshoot database",
                status: "PENDING_APPROVAL",
                type: "bb.task.general",
                instanceID: UNKNOWN_ID,
                databaseID: UNKNOWN_ID,
                statement: "",
                rollbackStatement: "",
              },
            ],
          },
        ],
        name: "Create database pipeline",
      },
      payload: {},
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
