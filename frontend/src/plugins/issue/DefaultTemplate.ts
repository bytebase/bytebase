import { IssueNew, UNKNOWN_ID } from "../../types";
import { IssueTemplate, TemplateContext } from "../types";

const template: IssueTemplate = {
  type: "bytebase.general",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueNew, "projectId" | "creatorId"> => {
    return {
      name: "",
      type: "bytebase.general",
      description: "",
      payload: {},
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
