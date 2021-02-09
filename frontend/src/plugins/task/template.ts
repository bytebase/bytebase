import { TaskTemplate, TemplateContext, Stage } from "../types";

export const taskTemplateList: TaskTemplate[] = [
  {
    type: "bytebase.datasource.create",
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
    stageListBuilder: (ctx: TemplateContext): Stage[] => {
      return [
        {
          name: "Request Data Source",
          type: "SIMPLE",
        },
      ];
    },
  },
];
