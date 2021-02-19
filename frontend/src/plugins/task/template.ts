import { TaskTemplate, TemplateContext } from "../types";
import { EnvironmentId, TaskType, TaskNew } from "../../types";

const allTaskTemplateList: TaskTemplate[] = [
  {
    type: "bytebase.general",
    buildTask: (ctx: TemplateContext): TaskNew => {
      return {
        type: "task",
        attributes: {
          name: "New General Task",
          type: "bytebase.general",
          content: "",
          stageProgressList: [
            {
              id: "1",
              name: "Request",
              type: "SIMPLE",
              status: "PENDING",
            },
          ],
          creator: {
            id: ctx.currentUser.id,
            name: ctx.currentUser.attributes.name,
          },
          payload: {},
        },
      };
    },
  },
  {
    type: "bytebase.datasource.create",
    buildTask: (ctx: TemplateContext): TaskNew => {
      return {
        type: "task",
        attributes: {
          name: "New Data Source",
          type: "bytebase.datasource.create",
          content: "Estimated QPS: 10",
          stageProgressList: [
            {
              id: "1",
              name: "Request Data Source",
              type: "SIMPLE",
              status: "PENDING",
            },
          ],
          creator: {
            id: ctx.currentUser.id,
            name: ctx.currentUser.attributes.name,
          },
          payload: {},
        },
      };
    },
    fieldList: [
      {
        category: "INPUT",
        id: 1,
        slug: "db",
        name: "Database Name",
        type: "String",
        required: true,
        preprocessor: (name: string): string => {
          // In case caller passes corrupted data.
          // Handled here instead of the caller, because it's
          // preprocessor specific behavior to handle fallback.
          return name?.toLowerCase();
        },
      },
      {
        category: "INPUT",
        id: 2,
        slug: "env",
        name: "Environment",
        type: "Environment",
        required: true,
        preprocessor: (environmentId: EnvironmentId): string => {
          // In case caller passes corrupted data.
          // Handled here instead of the caller, because it's
          // preprocessor specific behavior to handle fallback.
          return environmentId;
        },
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
  },
  {
    type: "bytebase.datasource.schema.update",
    buildTask: (ctx: TemplateContext): TaskNew => {
      return {
        type: "task",
        attributes: {
          name: "Update Schema",
          type: "bytebase.datasource.schema.update",
          content: "DDL: ",
          stageProgressList: ctx.environmentList.map((env) => {
            return {
              id: env.id,
              name: env.attributes.name,
              type: "ENVIRONMENT",
              status: "PENDING",
              runnable: {
                auto: true,
                run: () => {
                  console.log("Start", env.attributes.name);
                },
              },
            };
          }),
          creator: {
            id: ctx.currentUser.id,
            name: ctx.currentUser.attributes.name,
          },
          payload: {},
        },
      };
    },
    fieldList: [
      {
        category: "INPUT",
        id: 1,
        slug: "db",
        name: "Database Name",
        type: "String",
        required: true,
        preprocessor: (name: string): string => {
          // In case caller passes corrupted data.
          // Handled here instead of the caller, because it's
          // preprocessor specific behavior to handle fallback.
          return name?.toLowerCase();
        },
      },
    ],
  },
];

export function templateForType(type: TaskType): TaskTemplate | undefined {
  return allTaskTemplateList.find((template) => template.type === type);
}
