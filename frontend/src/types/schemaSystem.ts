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

export interface SchemaRule {
  id: string;
  category: CategoryType;
  engine: SchemaRuleEngineType;
  description: string;
  payload?: RulePayload;
  level: RuleLevel;
}

// export interface SelectedRule extends SchemaRule {
//   level: RuleLevel;
// }

// rule payload
interface NamingFormatPolicyPayload {
  format: string;
}

interface RequiredColumnsPolicyPayload {
  list: string[];
}

type PolicyPayload = NamingFormatPolicyPayload | RequiredColumnsPolicyPayload;

interface DatabaseSchemaRule {
  id: string;
  level: RuleLevel;
  payload?: PolicyPayload;
}

export const convertPolicyPayloadToRulePayload = (
  policyPayload: PolicyPayload | undefined,
  rulePayload: RulePayload | undefined
): RulePayload | undefined => {
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
  }, {} as RulePayload);
};

export const convertRulePayloadToPolicyPayload = (
  payload: RulePayload | undefined
): PolicyPayload | undefined => {
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
  environmentIdList: number[];
}

export interface DatabaseSchemaReviewPolicyCreate {
  // Domain specific fields
  name: string;
  ruleList: DatabaseSchemaRule[];
  environmentIdList: number[];
}

export type DatabaseSchemaReviewPolicyPatch = {
  // Domain specific fields
  name?: string;
  ruleList?: DatabaseSchemaRule[];
  environmentIdList?: number[];
};

interface RuleCategory {
  id: CategoryType;
  ruleList: SchemaRule[];
}

const categoryOrder: Map<CategoryType, number> = new Map([
  ["ENGINE", 5],
  ["NAMING", 4],
  ["QUERY", 3],
  ["TABLE", 2],
  ["COLUMN", 1],
]);

export const convertToCategoryList = (
  ruleList: SchemaRule[]
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

export interface SchemaReviewTemplate {
  name: string;
  imagePath: string;
  ruleList: SchemaRule[];
}

// TODO: i18n
export const ruleList: SchemaRule[] = [
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
