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

export interface RulePayload {
  [key: string]: StringPayload | StringArrayPayload | TemplatePayload;
}

export type SchemaRuleEngineType = "MYSQL" | "COMMON";

export type CategoryType = "ENGINE" | "NAMING" | "QUERY" | "TABLE" | "COLUMN";

export type RuleId =
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
  id: RuleId;
  category: CategoryType;
  engine: SchemaRuleEngineType;
  description: string;
  payload?: RulePayload;
  level: RuleLevel;
}

interface BaseSchemaRuleType {
  id: RuleId;
  level: RuleLevel;
}

interface NamingFormatRuleType extends BaseSchemaRuleType {
  format: string;
}

interface RequiredColumnRuleType extends BaseSchemaRuleType {
  columnList: string[];
}

type SchemaPolicyRule =
  | BaseSchemaRuleType
  | NamingFormatRuleType
  | RequiredColumnRuleType;

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
  environmentIdList: number[];
}

export interface DatabaseSchemaReviewPolicyCreate {
  // Domain specific fields
  name: string;
  ruleList: SchemaPolicyRule[];
  environmentIdList: number[];
}

export type DatabaseSchemaReviewPolicyPatch = {
  // Domain specific fields
  name?: string;
  ruleList?: SchemaPolicyRule[];
  environmentIdList?: number[];
};

interface RuleCategory {
  id: CategoryType;
  ruleList: RuleTemplate[];
}

export interface SchemaReviewTemplate {
  name: string;
  imagePath: string;
  ruleList: RuleTemplate[];
}

const categoryOrder: Map<CategoryType, number> = new Map([
  ["ENGINE", 5],
  ["NAMING", 4],
  ["QUERY", 3],
  ["TABLE", 2],
  ["COLUMN", 1],
]);

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

// TODO: i18n
export const ruleList: RuleTemplate[] = [
  {
    id: "engine.mysql.use-innodb",
    category: "ENGINE",
    engine: "MYSQL",
    description: "Require InnoDB as the storage engine.",
    level: RuleLevel.ERROR,
  },
  {
    id: "table.require-pk",
    category: "TABLE",
    engine: "COMMON",
    description: "Require the table to have a primary key.",
    level: RuleLevel.ERROR,
  },
  {
    id: "naming.table",
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the table name format. Default snake_lower_case.",
    level: RuleLevel.ERROR,

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
    level: RuleLevel.ERROR,

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
    level: RuleLevel.ERROR,

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
  },
  {
    id: "naming.index.uk",
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the unique key name format.",
    level: RuleLevel.ERROR,

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
  },
  {
    id: "naming.index.idx",
    category: "NAMING",
    engine: "COMMON",
    description: "Enforce the index name format.",
    level: RuleLevel.ERROR,

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
  },
  {
    id: "column.required",
    category: "COLUMN",
    engine: "COMMON",
    description: "Enforce the required columns in each table.",
    level: RuleLevel.ERROR,

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
    level: RuleLevel.ERROR,
  },
  {
    id: "query.select.no-select-all",
    category: "QUERY",
    engine: "COMMON",
    description: "Disallow 'SELECT *'.",
    level: RuleLevel.ERROR,
  },
  {
    id: "query.where.require",
    category: "QUERY",
    engine: "COMMON",
    description: "Require 'WHERE' clause.",
    level: RuleLevel.ERROR,
  },
  {
    id: "query.where.no-leading-wildcard-like",
    category: "QUERY",
    engine: "COMMON",
    description:
      "Disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.",
    level: RuleLevel.ERROR,
  },
];

export const convertPolicyRuleToRuleTemplate = (
  policyRule: SchemaPolicyRule,
  ruleTemplate: RuleTemplate
): RuleTemplate | undefined => {
  if (policyRule.id !== ruleTemplate.id) {
    return;
  }
  if (!ruleTemplate.payload) {
    return { ...ruleTemplate, level: policyRule.level };
  }

  const payload = Object.entries(ruleTemplate.payload).reduce(
    (obj, [key, val]) => {
      obj[key] = {
        ...val,
      };

      switch (val.type) {
        case PayloadType.STRING:
        case PayloadType.TEMPLATE:
          const format = (policyRule as NamingFormatRuleType).format;
          if (typeof format === "string") {
            obj[key].value = format;
          }
          break;
        case PayloadType.STRING_ARRAY:
          const list = (policyRule as RequiredColumnRuleType).columnList;
          if (Array.isArray(list)) {
            obj[key].value = list;
          }
          break;
      }

      return obj;
    },
    {} as RulePayload
  );

  return {
    ...ruleTemplate,
    level: policyRule.level,
    payload,
  };
};

export const convertRuleTemplateToPolicyRule = (
  rule: RuleTemplate
): SchemaPolicyRule => {
  const base: BaseSchemaRuleType = {
    id: rule.id,
    level: rule.level,
  };
  if (!rule.payload) {
    return base;
  }

  if ("format" in rule.payload) {
    return {
      ...base,
      format: rule.payload.format.value ?? rule.payload.format.default,
    } as NamingFormatRuleType;
  } else if ("list" in rule.payload) {
    return {
      ...base,
      columnList: rule.payload.list.value ?? rule.payload.list.default,
    } as RequiredColumnRuleType;
  } else {
    return base;
  }
};
