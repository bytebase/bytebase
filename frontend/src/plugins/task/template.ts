import {
  TaskField,
  TaskFieldId,
  TaskTemplate,
  TemplateContext,
} from "../types";
import { EnvironmentId, TaskType, TaskNew } from "../../types";

const DEFAULT_TEMPLATE = {
  type: "bytebase.general",
  buildTask: (ctx: TemplateContext): TaskNew => {
    return {
      name: "New General Task",
      type: "bytebase.general",
      description: "",
      stageProgressList: [
        {
          id: "1",
          name: "Request",
          type: "SIMPLE",
          status: "PENDING",
        },
      ],
      creatorId: ctx.currentUser.id,
      payload: {},
    };
  },
  fieldList: [],
};

const allTaskTemplateList: TaskTemplate[] = [
  DEFAULT_TEMPLATE,
  {
    type: "bytebase.database.request",
    buildTask: (ctx: TemplateContext): TaskNew => {
      return {
        name: "Request new database",
        type: "bytebase.database.request",
        description: "Estimated QPS: 10",
        stageProgressList: [
          {
            id: "1",
            name: "Request Data Source",
            type: "SIMPLE",
            status: "PENDING",
          },
        ],
        creatorId: ctx.currentUser.id,
        payload: {},
      };
    },
    fieldList: [
      {
        category: "INPUT",
        id: 1,
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
        category: "INPUT",
        id: 2,
        slug: "db",
        name: "DB Name",
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
    type: "bytebase.database.schema.update",
    buildTask: (ctx: TemplateContext): TaskNew => {
      return {
        name: "Update Schema",
        type: "bytebase.database.schema.update",
        description: "DDL: ",
        stageProgressList: ctx.environmentList.map((env) => {
          return {
            id: env.id,
            name: env.name,
            type: "ENVIRONMENT",
            status: "PENDING",
            runnable: {
              auto: true,
              run: () => {
                console.log("Start", env.name);
              },
            },
          };
        }),
        creatorId: ctx.currentUser.id,
        payload: {},
      };
    },
    fieldList: [
      {
        category: "INPUT",
        id: 1,
        slug: "db",
        name: "DB Name",
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

export function defaulTemplate(): TaskTemplate {
  return DEFAULT_TEMPLATE;
}

export function templateForType(type: TaskType): TaskTemplate | undefined {
  return allTaskTemplateList.find((template) => template.type === type);
}

export function fieldFromId(
  template: TaskTemplate,
  fieldId: TaskFieldId
): TaskField | undefined {
  if (template.fieldList) {
    return template.fieldList.find((field) => field.id == fieldId);
  }
  return undefined;
}
