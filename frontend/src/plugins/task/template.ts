import { TaskTemplate } from "../types";

export const taskTemplateList: TaskTemplate[] = [
  {
    type: "bytebase.datasource.create",
    fieldList: [
      {
        id: 1,
        slug: "db",
        name: "Database Name",
        type: "String",
        required: true,
        preprocessor: (name: string): string => {
          return name.toLowerCase();
        },
      },
    ],
  },
];
