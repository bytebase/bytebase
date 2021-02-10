import { TaskTemplate, TemplateContext, Stage } from "../types";
import { TaskNew } from "../../types";

export const taskTemplateList: TaskTemplate[] = [
  {
    type: "bytebase.general",
    buildTask: (ctx: TemplateContext): TaskNew => {
      return {
        type: "task",
        attributes: {
          name: "New General Task",
          type: "bytebase.general",
          content: "blablabla",
          stageProgressList: [
            {
              id: 1,
              name: "Request",
              type: "SIMPLE",
              status: "CREATED",
            },
          ],
          creator: {
            id: ctx.currentUser.id,
            name: ctx.currentUser.attributes.name,
          },
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
          content: "blablabla",
          stageProgressList: [
            {
              id: 1,
              name: "Request Data Source",
              type: "ENVIRONMENT",
              status: "CREATED",
            },
          ],
          creator: {
            id: ctx.currentUser.id,
            name: ctx.currentUser.attributes.name,
          },
        },
      };
    },
    outputFieldList: [
      {
        id: 1,
        name: "Data Source URL1",
        required: true,
      },
      {
        id: 2,
        name: "Hello world",
        required: true,
      },
      {
        id: 3,
        name: "Data Source URL3",
        required: true,
      },
    ],
    fieldList: [
      {
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
