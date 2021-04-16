import isEmpty from "lodash-es/isEmpty";

import {
  TaskTemplate,
  TemplateContext,
  TaskBuiltinFieldId,
  TaskContext,
} from "../types";

import { Database, Stage, StageNew, TaskNew, UNKNOWN_ID } from "../../types";

const template: TaskTemplate = {
  type: "bytebase.database.schema.update",
  buildTask: (ctx: TemplateContext): TaskNew => {
    const payload: any = {};
    return {
      name: "Change db",
      type: "bytebase.database.schema.update",
      description: "",
      stageList: ctx.databaseList.map(
        (database: Database): StageNew => {
          return {
            id: "1",
            name: `[${database.instance.environment.name}] ${database.name}`,
            type: "bytebase.stage.schema.update",
            databaseId: database.id,
          };
        }
      ),
      creatorId: ctx.currentUser.id,
      subscriberIdList: [],
      payload,
    };
  },
  fieldList: [],
};

export default template;
