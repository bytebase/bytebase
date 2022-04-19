import { SchemaReviewPolicyId } from "./id";
import { Principal } from "./principal";

export enum RuleLevel {
  DISABLED = "DISABLED",
  ERROR = "ERROR",
  WARNING = "WARNING",
}

export const LEVEL_LIST = [
  RuleLevel.ERROR,
  RuleLevel.WARNING,
  RuleLevel.DISABLED,
];

export enum PayloadType {
  STRING = "STRING",
  STRING_ARRAY = "STRING[]",
  TEMPLATE = "TEMPLATE",
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

interface NamingFormatRulePayload {
  format: StringPayload | TemplatePayload;
}

interface RequiredColumnsRulePayload {
  list: StringArrayPayload;
}

export type RulePayload =
  | NamingFormatRulePayload
  | RequiredColumnsRulePayload
  | undefined;

export type SchemaRuleEngineType = "MYSQL" | "COMMON";

export type CategoryType = "ENGINE" | "NAMING" | "QUERY" | "TABLE" | "COLUMN";

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

// rule payload
interface NamingFormatPolicyPayload {
  format: string;
}

interface RequiredColumnsPolicyPayload {
  list: string[];
}

type PolicyPayload =
  | NamingFormatPolicyPayload
  | RequiredColumnsPolicyPayload
  | undefined;

interface DatabaseSchemaRule {
  id: string;
  level: RuleLevel;
  payload?: PolicyPayload;
}

export const convertPolicyPayloadToRulePayload = (
  policyPayload: PolicyPayload,
  rulePayload: RulePayload
): RulePayload => {
  if (!policyPayload || !rulePayload) {
    return;
  }

  return Object.entries(rulePayload).reduce((obj, [key, val]) => {
    obj[key] = {
      ...val,
    };

    switch (val.type) {
      case PayloadType.STRING:
      case PayloadType.TEMPLATE:
        const format = (policyPayload as NamingFormatPolicyPayload).format;
        if (typeof format === "string") {
          obj[key].value = format;
        }
        break;
      case PayloadType.STRING_ARRAY:
        const list = (policyPayload as RequiredColumnsPolicyPayload).list;
        if (Array.isArray(list)) {
          obj[key].value = list;
        }
        break;
    }

    return obj;
  }, {} as any);
};

export const convertRulePayloadToPolicyPayload = (
  payload: RulePayload
): PolicyPayload => {
  if (!payload) {
    return;
  }
  if ("format" in payload) {
    return {
      format: payload.format.value ?? payload.format.default,
    } as NamingFormatPolicyPayload;
  } else if ("list" in payload) {
    return {
      list: payload.list.value ?? payload.list.default,
    } as RequiredColumnsPolicyPayload;
  }
};

export interface DatabaseSchemaReviewPolicy {
  id: SchemaReviewPolicyId;

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
  ruleList: T[];
}

const categoryOrder: Map<CategoryType, number> = new Map([
  ["ENGINE", 5],
  ["NAMING", 4],
  ["QUERY", 3],
  ["TABLE", 2],
  ["COLUMN", 1],
]);

export function convertToCategoryList<T extends SchemaRule>(
  ruleList: T[]
): RuleCategory<T>[] {
  const dict = ruleList.reduce((dict, rule) => {
    if (!dict[rule.category]) {
      const id = rule.category.toLowerCase();
      dict[rule.category] = {
        id: rule.category,
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
    category: "ENGINE",
    engine: "MYSQL",
    description: "Require InnoDB as the storage engine.",
  },
  {
    id: "table.require-pk",
    category: "TABLE",
    engine: "COMMON",
    description: "Require the table to have a primary key.",
  },
  {
    id: "naming.table",
    category: "NAMING",
    engine: "COMMON",
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
    category: "NAMING",
    engine: "COMMON",
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
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the primary key name format.",
    payload: {
      format: {
        type: PayloadType.TEMPLATE,
        default: "^pk_{{table}}_{{column_list}}$",
        templateList: [
          {
            id: "TABLE",
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
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the unique key name format.",
    payload: {
      format: {
        type: PayloadType.TEMPLATE,
        default: "^uk_{{table}}_{{column_list}}$",
        templateList: [
          {
            id: "TABLE",
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
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the index name format.",
    payload: {
      format: {
        type: PayloadType.TEMPLATE,
        default: "^idx_{{table}}_{{column_list}}$",
        templateList: [
          {
            id: "TABLE",
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
    category: "COLUMN",
    engine: "COMMON",
    description: "Enforce the required columns in each table.",
    payload: {
      list: {
        type: PayloadType.STRING_ARRAY,
        default: ["id", "created_ts", "updated_ts", "creator_id", "updater_id"],
      },
    },
  },
  {
    id: "column.no-null",
    category: "COLUMN",
    engine: "COMMON",
    description: "Columns cannot have NULL value.",
  },
  {
    id: "query.select.no-select-all",
    category: "QUERY",
    engine: "COMMON",
    description: "Disallow 'SELECT *'.",
  },
  {
    id: "query.where.require",
    category: "QUERY",
    engine: "COMMON",
    description: "Require 'WHERE' clause.",
  },
  {
    id: "query.where.no-leading-wildcard-like",
    category: "QUERY",
    engine: "COMMON",
    description:
      "Disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.",
  },
];
