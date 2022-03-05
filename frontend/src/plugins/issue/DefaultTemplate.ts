import { IssueCreate, UNKNOWN_ID } from "../../types";
import { IssueTemplate } from "../types";

const template: IssueTemplate = {
  type: "bb.issue.general",
  buildIssue: (/* ctx: TemplateContext */): Omit<
    IssueCreate,
    "projectId" | "creatorId"
  > => {
    return {
      name: "",
      type: "bb.issue.general",
      description: "",
      assigneeId: UNKNOWN_ID,
      pipeline: {
        stageList: [
          {
            name: "Troubleshoot database",
            environmentId: UNKNOWN_ID,
            taskList: [
              {
                name: "Troubleshoot database",
                status: "PENDING_APPROVAL",
                type: "bb.task.general",
                instanceId: UNKNOWN_ID,
                databaseId: UNKNOWN_ID,
                statement: "",
                earliestAllowedTs: 0,
              },
            ],
          },
        ],
        name: "Create database pipeline",
      },
      createContext: {},
      payload: {},
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
