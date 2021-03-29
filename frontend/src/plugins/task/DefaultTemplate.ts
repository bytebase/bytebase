import isEmpty from "lodash-es/isEmpty";
import { DatabaseId, EnvironmentId, TaskNew, UNKNOWN_ID } from "../../types";
import { TaskBuiltinFieldId, TaskTemplate, TemplateContext } from "../types";

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
  fieldList: [
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.ENVIRONMENT,
      slug: "environment",
      name: "Environment",
      type: "Environment",
      required: true,
      isEmpty: (value: EnvironmentId): boolean => {
        return isEmpty(value);
      },
    },
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.DATABASE,
      slug: "database",
      name: "Database",
      type: "Database",
      required: true,
      isEmpty: (value: DatabaseId): boolean => {
        return value == undefined || value == UNKNOWN_ID;
      },
    },
  ],
};

export default template;
