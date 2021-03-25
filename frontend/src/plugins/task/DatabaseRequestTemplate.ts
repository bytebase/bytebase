import isEmpty from "lodash-es/isEmpty";
import {
  TaskField,
  TaskTemplate,
  TemplateContext,
  TaskBuiltinFieldId,
  TaskFieldReferenceProviderContext,
  DatabaseFieldPayload,
} from "../types";
import { linkfy, validLink } from "../../utils";
import { Task, TaskNew, EnvironmentId } from "../../types";

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
      subscriberIdList: [],
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
      isEmpty: (value: EnvironmentId): boolean => {
        return isEmpty(value);
      },
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
      id: "99",
      slug: "datasource",
      name: "Data source",
      type: "String",
      required: true,
      isEmpty: (value: string): boolean => {
        return isEmpty(value?.trim());
      },
      provider: ({ task, field }: { task: Task; field: TaskField }) => {
        const currentValue = task.payload[field.id];
        if (validLink(currentValue)) {
          return {
            title: "view data source",
            link: linkfy(currentValue),
          };
        }

        let title = "create data source";
        let link = "/db/new";
        const databasePayload: DatabaseFieldPayload =
          task.payload[TaskBuiltinFieldId.DATABASE];
        if (!databasePayload.isNew) {
          title = "assign data source";
        }

        const queryParamList: string[] = [];

        const environmentId = task.payload[TaskBuiltinFieldId.ENVIRONMENT];
        if (environmentId) {
          queryParamList.push(`environment=${environmentId}`);
        }

        if (databasePayload.name) {
          queryParamList.push(`name=${databasePayload.name}`);
        }

        // If we are creating a new database, we always assign RW to the owner.
        if (!databasePayload.isNew && databasePayload.readOnly) {
          queryParamList.push(`readonly=true`);
        }

        queryParamList.push(`owner=${task.creator.id}`);

        queryParamList.push(`task=${task.id}`);

        link += "?" + queryParamList.join("&");

        return {
          title,
          link,
        };
      },
    },
  ],
};

export default template;
