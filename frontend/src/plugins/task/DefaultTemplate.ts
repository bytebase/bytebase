import isEmpty from "lodash-es/isEmpty";
import { TaskNew, UNKNOWN_ID } from "../../types";
import {
  TaskBuiltinFieldId,
  TaskContext,
  TaskTemplate,
  TemplateContext,
} from "../types";

const template: TaskTemplate = {
  type: "bytebase.general",
  buildTask: (ctx: TemplateContext): TaskNew => {
    return {
      name: "",
      type: "bytebase.general",
      description: "",
      stageList: [
        {
          name: "Request",
          type: "bytebase.stage.general",
          databaseId: UNKNOWN_ID,
          stepList: [],
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
