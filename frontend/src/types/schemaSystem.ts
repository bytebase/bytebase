import { SchemaReviewId } from "./id";
import { Principal } from "./principal";

export enum RuleLevel {
  DISABLED = "disabled",
  ERROR = "error",
  WARNING = "warning",
}

export const LEVEL_LIST = [
  RuleLevel.ERROR,
  RuleLevel.WARNING,
  RuleLevel.DISABLED,
];

enum PayloadType {
  STRING = "string",
  STRING_ARRAY = "string[]",
  TEMPLATE = "template",
}

interface StringPayload {
  type: PayloadType.STRING;
  default: string;
  value?: string;
}

interface StringArrayPayload {
  type: PayloadType.STRING_ARRAY;
  default: string[];
  value?: string[];
}

interface TemplatePayload {
  type: PayloadType.TEMPLATE;
  default: string;
  templateList: { id: string; description?: string }[];
  value?: string;
}

export interface RulePayload {
  [key: string]: StringPayload | StringArrayPayload | TemplatePayload;
}

export type SchemaRuleEngineType = "MySQL" | "Common";

export type CategoryType = "engine" | "naming" | "query" | "table" | "column";

export interface SchemaRule {
  id: string;
  category: CategoryType;
  engine: SchemaRuleEngineType;
  description: string;
  payload?: RulePayload;
}

export interface SelectedRule extends SchemaRule {
  level: RuleLevel;
}

interface DatabaseSchemaRule {
  id: string;
  level: RuleLevel;
  payload?: {
    [key: string]: any;
  };
}

export interface DatabaseSchemaReviewPolicy {
  id: SchemaReviewId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  ruleList: DatabaseSchemaRule[];
  environmentList: number[];
}

export interface DatabaseSchemaReviewPolicyCreate {
  // Domain specific fields
  name: string;
  ruleList: DatabaseSchemaRule[];
  environmentList: number[];
}

export type DatabaseSchemaReviewPolicyPatch = {
  // Domain specific fields
  name?: string;
  ruleList?: DatabaseSchemaRule[];
  environmentList?: number[];
};

interface RuleCategory<T extends SchemaRule> {
  id: CategoryType;
  name: string;
  ruleList: T[];
}

const categoryOrder: Map<CategoryType, number> = new Map([
  ["engine", 5],
  ["naming", 4],
  ["query", 3],
  ["table", 2],
  ["column", 1],
]);

export function convertToCategoryList<T extends SchemaRule>(
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

export interface SchemaReviewTemplate {
  name: string;
  imagePath: string;
  ruleList: SelectedRule[];
}

// TODO: i18n
export const ruleList: SchemaRule[] = [
  {
    id: "engine.mysql.use-innodb",
    category: "engine",
    engine: "MySQL",
    description: "Require InnoDB as the storage engine.",
  },
  {
    id: "table.require-pk",
    category: "table",
    engine: "Common",
    description: "Require the table to have a primary key.",
  },
  {
    id: "naming.table",
    category: "naming",
    engine: "Common",
    description: "Enforce the table name format. Default snake_lower_case.",
    payload: {
      format: {
        type: PayloadType.STRING,
        default: "^[a-z]+(_[a-z]+)?$",
      },
    },
  },
  {
    id: "naming.column",
    category: "naming",
    engine: "Common",
    description: "Enforce the column name format. Default snake_lower_case.",
    payload: {
      format: {
        type: PayloadType.STRING,
        default: "^[a-z]+(_[a-z]+)?$",
      },
    },
  },
  {
    id: "naming.index.pk",
    category: "naming",
    engine: "Common",
    description: "Enforce the primary key name format.",
    payload: {
      pk: {
        type: PayloadType.TEMPLATE,
        default: "^pk_{{table}}_{{column_list}}$",
        templateList: [
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
    engine: "Common",
    description: "Enforce the unique key name format.",
    payload: {
      uk: {
        type: PayloadType.TEMPLATE,
        default: "^uk_{{table}}_{{column_list}}$",
        templateList: [
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
    engine: "Common",
    description: "Enforce the index name format.",
    payload: {
      idx: {
        type: PayloadType.TEMPLATE,
        default: "^idx_{{table}}_{{column_list}}$",
        templateList: [
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
    engine: "Common",
    description: "Enforce the required columns in each table.",
    payload: {
      columns: {
        type: PayloadType.STRING_ARRAY,
        default: ["id", "created_ts", "updated_ts", "creator_id", "updater_id"],
      },
    },
  },
  {
    id: "column.no-null",
    category: "column",
    engine: "Common",
    description: "Columns cannot have NULL value.",
  },
  {
    id: "query.select.no-select-all",
    category: "query",
    engine: "Common",
    description: "Disallow 'SELECT *'.",
  },
  {
    id: "query.where.require",
    category: "query",
    engine: "Common",
    description: "Require 'WHERE' clause.",
  },
  {
    id: "query.where.no-leading-wildcard-like",
    category: "query",
    engine: "Common",
    description:
      "Disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.",
  },
];
