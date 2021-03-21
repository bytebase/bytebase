import isEmpty from "lodash-es/isEmpty";
import {
  TaskTemplate,
  TemplateContext,
  TaskBuiltinFieldId,
  DatabaseFieldPayload,
} from "../types";

import { TaskNew } from "../../types";

const template: TaskTemplate = {
  type: "bytebase.database.request",
  buildTask: (ctx: TemplateContext): TaskNew => {
    const payload: any = {};
    if (ctx.environmentList.length > 0) {
      // Set the last element as the default value.
      // Normally the last environment is the prod env and is most commonly used.
      payload[TaskBuiltinFieldId.ENVIRONMENT] =
        ctx.environmentList[ctx.environmentList.length - 1].id;
    }
    payload[TaskBuiltinFieldId.DATABASE] = {
      isNew: true,
      name: "",
      // Set read-only defaults to true since only read access is needed most of the time
      // and sticks to the least privilege rule.
      readOnly: true,
    };
    return {
      name: "Request new database",
      type: "bytebase.database.request",
      description: "Estimated QPS: 10",
      stageProgressList: [
        {
          id: "1",
          name: "Request database",
          type: "SIMPLE",
          status: "PENDING",
        },
      ],
      creatorId: ctx.currentUser.id,
      payload,
    };
  },
  fieldList: [
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.ENVIRONMENT,
      slug: "env",
      name: "Environment",
      type: "Environment",
      required: true,
    },
    {
      category: "INPUT",
      id: TaskBuiltinFieldId.DATABASE,
      slug: "db",
      name: "DB name",
      type: "NewDatabase",
      required: true,
      isEmpty: (value: DatabaseFieldPayload): boolean => {
        if (value.isNew) {
          return isEmpty(value.name);
        }
        return isEmpty(value.id);
      },
      placeholder: "New database name",
    },
    {
      category: "OUTPUT",
      id: 99,
      slug: "datasource",
      name: "Data Source URL",
      type: "String",
      required: true,
    },
  ],
};

export default template;
