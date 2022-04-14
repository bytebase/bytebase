export enum RuleLevel {
  Disabled = "disabled",
  Error = "error",
  Warning = "warning",
}

export const levelList = [
  { id: RuleLevel.Error, name: "Error" },
  { id: RuleLevel.Warning, name: "Warning" },
  { id: RuleLevel.Disabled, name: "Disabled" },
];

enum PayloadType {
  String = "string",
  StringArray = "string[]",
  Template = "template",
}

interface StringPayload {
  type: PayloadType.String;
  default: string;
  value?: string;
}

interface StringArrayPayload {
  type: PayloadType.StringArray;
  default: string[];
  value?: string[];
}

interface TemplatePayload {
  type: PayloadType.Template;
  default: string;
  templates: { id: string; description?: string }[];
  value?: string;
}

export interface RulePayload {
  [key: string]: StringPayload | StringArrayPayload | TemplatePayload;
}

export type DatabaseType = "MySQL" | "Common";

export type CategoryType = "engine" | "naming" | "query" | "table" | "column";

export const categoryOrder: Map<CategoryType, number> = new Map([
  ["engine", 5],
  ["naming", 4],
  ["query", 3],
  ["table", 2],
  ["column", 1],
]);

export interface Rule {
  id: string;
  category: CategoryType;
  database: DatabaseType[];
  description: string;
  payload?: RulePayload;
}

export interface SelectedRule extends Rule {
  level: RuleLevel;
}

interface SchemaRule {
  id: string;
  level: RuleLevel;
  payload?: {
    [key: string]: any;
  };
}

export interface DatabaseSchemaGuide {
  id: number;
  name: string;
  ruleList: SchemaRule[];
  environmentList: number[];
  createdTs: number;
  updatedTs: number;
}

export interface RuleCategory<T extends Rule> {
  id: CategoryType;
  name: string;
  ruleList: T[];
}

export function convertToCategoryList<T extends Rule>(
  ruleList: T[]
): RuleCategory<T>[] {
  const dict = ruleList.reduce((dict, rule) => {
    if (!dict[rule.category]) {
      const id = rule.category.toLowerCase();
      const name = `${id[0].toUpperCase()}${id.slice(1)}`;
      dict[rule.category] = {
        id: rule.category,
        name,
        ruleList: [],
      };
    }
    dict[rule.category].ruleList.push(rule);
    return dict;
  }, {} as { [key: string]: RuleCategory<T> });

  return Object.values(dict).sort(
    (c1, c2) =>
      (categoryOrder.get(c2.id as CategoryType) || 0) -
      (categoryOrder.get(c1.id as CategoryType) || 0)
  );
}

export const ruleList: Rule[] = [
  {
    id: "engine.mysql.use-innodb",
    category: "engine",
    database: ["MySQL"],
    description: "Require InnoDB as the storage engine.",
  },
  {
    id: "table.require-pk",
    category: "table",
    database: ["Common"],
    description: "Require the table to have a primary key.",
  },
  {
    id: "naming.table",
    category: "naming",
    database: ["Common"],
    description: "Enforce the table name format. Default snake_lower_case.",
    payload: {
      format: {
        type: PayloadType.String,
        default: "^[a-z]+(_[a-z]+)?$",
      },
    },
  },
  {
    id: "naming.column",
    category: "naming",
    database: ["Common"],
    description: "Enforce the column name format. Default snake_lower_case.",
    payload: {
      format: {
        type: PayloadType.String,
        default: "^[a-z]+(_[a-z]+)?$",
      },
    },
  },
  {
    id: "naming.index.pk",
    category: "naming",
    database: ["Common"],
    description: "Enforce the primary key name format.",
    payload: {
      pk: {
        type: PayloadType.Template,
        default: "^pk_{{table}}_{{column_list}}$",
        templates: [
          {
            id: "table",
            description: "The table name",
          },
          {
            id: "column_list",
            description: "Index column names, joined by _",
          },
        ],
      },
    },
  },
  {
    id: "naming.index.uk",
    category: "naming",
    database: ["Common"],
    description: "Enforce the unique key name format.",
    payload: {
      uk: {
        type: PayloadType.Template,
        default: "^uk_{{table}}_{{column_list}}$",
        templates: [
          {
            id: "table",
            description: "The table name",
          },
          {
            id: "column_list",
            description: "Index column names, joined by _",
          },
        ],
      },
    },
  },
  {
    id: "naming.index.idx",
    category: "naming",
    database: ["Common"],
    description: "Enforce the index name format.",
    payload: {
      idx: {
        type: PayloadType.Template,
        default: "^idx_{{table}}_{{column_list}}$",
        templates: [
          {
            id: "table",
            description: "The table name",
          },
          {
            id: "column_list",
            description: "Index column names, joined by _",
          },
        ],
      },
    },
  },
  {
    id: "column.required",
    category: "column",
    database: ["Common"],
    description: "Enforce the required columns in each table.",
    payload: {
      columns: {
        type: PayloadType.StringArray,
        default: ["id", "created_ts", "updated_ts", "creator_id", "updater_id"],
      },
    },
  },
  {
    id: "column.no-null",
    category: "column",
    database: ["Common"],
    description: "Columns cannot have NULL value.",
  },
  {
    id: "query.select.no-select-all",
    category: "query",
    database: ["Common"],
    description: "Disallow 'SELECT *'.",
  },
  {
    id: "query.where.require",
    category: "query",
    database: ["Common"],
    description: "Require 'WHERE' clause.",
  },
  {
    id: "query.where.no-leading-wildcard-like",
    category: "query",
    database: ["Common"],
    description:
      "Disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.",
  },
];
