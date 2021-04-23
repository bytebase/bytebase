import { IssueNew, UNKNOWN_ID } from "../../types";
import { IssueTemplate, TemplateContext } from "../types";

const template: IssueTemplate = {
  type: "bb.general",
  buildIssue: (
    ctx: TemplateContext
  ): Omit<IssueNew, "projectId" | "creatorId"> => {
    return {
      name: "",
      type: "bb.general",
      description: "",
      payload: {},
    };
  },
  inputFieldList: [],
  outputFieldList: [],
};

export default template;
