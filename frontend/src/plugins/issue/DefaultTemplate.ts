import { IssueNew, UNKNOWN_ID } from "../../types";
import { IssueTemplate, TemplateContext } from "../types";

const template: IssueTemplate = {
  type: "bytebase.general",
  buildIssue: (ctx: TemplateContext): IssueNew => {
    return {
      name: "",
      type: "bytebase.general",
      description: "",
      stageList: [],
      creatorId: ctx.currentUser.id,
      subscriberIdList: [],
      payload: {},
    };
  },
  fieldList: [],
};

export default template;
