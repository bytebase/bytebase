import { IssueCreate, UNKNOWN_ID } from "../../types";
import { IssueTemplate } from "../types";

const template: IssueTemplate = {
  type: "bb.issue.grant.request",
  buildIssue: (): Omit<IssueCreate, "projectId" | "creatorId"> => {
    return {
      name: "",
      type: "bb.issue.grant.request",
      description: "",
      assigneeId: UNKNOWN_ID,
      createContext: {},
      payload: {},
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
