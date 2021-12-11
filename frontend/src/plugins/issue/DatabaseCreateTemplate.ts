import { IssueCreate, UNKNOWN_ID } from "../../types";
import { IssueTemplate, TemplateContext } from "../types";

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
      assigneeId: UNKNOWN_ID,
      pipeline: {
        stageList: [],
        name: "",
      },
      createContext: {},
      payload,
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
