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

enum PayloadType {
  STRING = "STRING",
  STRING_ARRAY = "STRING_ARRAY",
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

interface NamingFormatRuleTemplatePayload {
  format: StringPayload | TemplatePayload;
}

interface RequiredColumnRuleTemplatePayload {
  list: StringArrayPayload;
}

export type RuleTemplatePayload =
  | NamingFormatRuleTemplatePayload
  | RequiredColumnRuleTemplatePayload
  | undefined;

export type SchemaRuleEngineType = "MYSQL" | "COMMON";

export type CategoryType = "ENGINE" | "NAMING" | "QUERY" | "TABLE" | "COLUMN";

export type RuleType =
  | "engine.mysql.use-innodb"
  | "table.require-pk"
  | "naming.table"
  | "naming.column"
  | "naming.index.pk"
  | "naming.index.uk"
  | "naming.index.idx"
  | "column.required"
  | "column.no-null"
  | "query.select.no-select-all"
  | "query.where.require"
  | "query.where.no-leading-wildcard-like";

export interface RuleTemplate {
  type: RuleType;
  category: CategoryType;
  engine: SchemaRuleEngineType;
  description: string;
  payload?: RuleTemplatePayload;
  level: RuleLevel;
}

interface NamingFormatPayload {
  format: string;
}

interface RequiredColumnPayload {
  columnList: string[];
}

export interface SchemaPolicyRule {
  type: RuleType;
  level: RuleLevel;
  payload?: NamingFormatPayload | RequiredColumnPayload;
}

export interface DatabaseSchemaReviewPolicy {
  id: SchemaReviewPolicyId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  ruleList: SchemaPolicyRule[];
  environmentId?: number;
}

export interface DatabaseSchemaReviewPolicyCreate {
  // Domain specific fields
  name: string;
  ruleList: SchemaPolicyRule[];
  environmentId?: number;
}

export type DatabaseSchemaReviewPolicyPatch = {
  // Domain specific fields
  name?: string;
  ruleList?: SchemaPolicyRule[];
  environmentId?: number;
};

export interface SchemaReviewPolicyTemplate {
  name: string;
  imagePath: string;
  ruleList: RuleTemplate[];
}

// TODO: i18n
export const ruleTemplateList: RuleTemplate[] = [
  {
    type: "engine.mysql.use-innodb",
    category: "ENGINE",
    engine: "MYSQL",
    description: "Require InnoDB as the storage engine.",
    level: RuleLevel.ERROR,
  },
  {
    type: "table.require-pk",
    category: "TABLE",
    engine: "COMMON",
    description: "Require the table to have a primary key.",
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.table",
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the table name format. Default snake_lower_case.",
    payload: {
      format: {
        type: PayloadType.STRING,
        default: "^[a-z]+(_[a-z]+)?$",
      },
    },
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.column",
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the column name format. Default snake_lower_case.",
    payload: {
      format: {
        type: PayloadType.STRING,
        default: "^[a-z]+(_[a-z]+)?$",
      },
    },
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.index.pk",
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the primary key name format.",
    payload: {
      format: {
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
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.index.uk",
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the unique key name format.",
    payload: {
      format: {
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
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.index.idx",
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the index name format.",
    payload: {
      format: {
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
    level: RuleLevel.ERROR,
  },
  {
    type: "column.required",
    category: "COLUMN",
    engine: "COMMON",
    description: "Enforce the required columns in each table.",
    payload: {
      list: {
        type: PayloadType.STRING_ARRAY,
        default: ["id", "created_ts", "updated_ts", "creator_id", "updater_id"],
      },
    },
    level: RuleLevel.ERROR,
  },
  {
    type: "column.no-null",
    category: "COLUMN",
    engine: "COMMON",
    description: "Columns cannot have NULL value.",
    level: RuleLevel.ERROR,
  },
  {
    type: "query.select.no-select-all",
    category: "QUERY",
    engine: "COMMON",
    description: "Disallow 'SELECT *'.",
    level: RuleLevel.ERROR,
  },
  {
    type: "query.where.require",
    category: "QUERY",
    engine: "COMMON",
    description: "Require 'WHERE' clause.",
    level: RuleLevel.ERROR,
  },
  {
    type: "query.where.no-leading-wildcard-like",
    category: "QUERY",
    engine: "COMMON",
    description:
      "Disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.",
    level: RuleLevel.ERROR,
  },
];

const categoryOrder: Map<CategoryType, number> = new Map([
  ["ENGINE", 5],
  ["NAMING", 4],
  ["QUERY", 3],
  ["TABLE", 2],
  ["COLUMN", 1],
]);

interface RuleCategory {
  id: CategoryType;
  ruleList: RuleTemplate[];
}

export const convertToCategoryList = (
  ruleList: RuleTemplate[]
): RuleCategory[] => {
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
  }, {} as { [key: string]: RuleCategory });

  return Object.values(dict).sort(
    (c1, c2) =>
      (categoryOrder.get(c2.id as CategoryType) || 0) -
      (categoryOrder.get(c1.id as CategoryType) || 0)
  );
};

export const convertPolicyRuleToRuleTemplate = (
  policyRule: SchemaPolicyRule,
  ruleTemplate: RuleTemplate
): RuleTemplate => {
  const res = { ...ruleTemplate, level: policyRule.level };

  if (!ruleTemplate.payload) {
    return res;
  }

  switch (ruleTemplate.type) {
    case "naming.column":
    case "naming.index.idx":
    case "naming.index.pk":
    case "naming.index.uk":
    case "naming.table":
      const namingPayload =
        ruleTemplate.payload as NamingFormatRuleTemplatePayload;
      namingPayload.format.value = (
        policyRule.payload as NamingFormatPayload
      ).format;
      return {
        ...res,
        payload: namingPayload,
      };
    case "column.required":
      const requiredColumnPayload =
        ruleTemplate.payload as RequiredColumnRuleTemplatePayload;
      requiredColumnPayload.list.value = (
        policyRule.payload as RequiredColumnPayload
      ).columnList;
      return {
        ...res,
        payload: requiredColumnPayload,
      };
  }

  return res;
};

export const convertRuleTemplateToPolicyRule = (
  rule: RuleTemplate
): SchemaPolicyRule => {
  const base: SchemaPolicyRule = {
    type: rule.type,
    level: rule.level,
  };
  if (!rule.payload) {
    return base;
  }

  switch (rule.type) {
    case "naming.column":
    case "naming.index.idx":
    case "naming.index.pk":
    case "naming.index.uk":
    case "naming.table":
      const namingPayload = rule.payload as NamingFormatRuleTemplatePayload;
      const format = namingPayload.format.value ?? namingPayload.format.default;
      return {
        ...base,
        payload: {
          format,
        },
      };
    case "column.required":
      const requiredColumnPayload =
        rule.payload as RequiredColumnRuleTemplatePayload;
      const columnList =
        requiredColumnPayload.list.value ?? requiredColumnPayload.list.default;
      return {
        ...base,
        payload: {
          columnList,
        },
      };
  }

  return base;
};
