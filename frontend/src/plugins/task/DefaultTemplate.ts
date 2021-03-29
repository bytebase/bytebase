import { TaskNew } from "../../types";
import { TaskTemplate, TemplateContext } from "../types";

const template: TaskTemplate = {
  type: "bytebase.general",
  buildTask: (ctx: TemplateContext): TaskNew => {
    return {
      name: "",
      type: "bytebase.general",
      description: "",
      stageList: [
        {
          id: "1",
          name: "Request",
          type: "bytebase.stage.general",
          status: "PENDING",
        },
      ],
      creatorId: ctx.currentUser.id,
      subscriberIdList: [],
      payload: {},
    };
  },
  fieldList: [],
};

export default template;
