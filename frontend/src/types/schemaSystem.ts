import { useI18n } from "vue-i18n";
import { PolicyId } from "./id";
import { Principal } from "./principal";
import { RowStatus } from "./common";
import { Environment } from "./environment";

// The engine type for rule template
export type SchemaRuleEngineType = "MYSQL" | "COMMON";

// The category type for rule template
export type CategoryType =
  | "ENGINE"
  | "NAMING"
  | "STATEMENT"
  | "TABLE"
  | "COLUMN"
  | "SCHEMA";

// The rule level
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

// StringPayload is the string type payload configuration options and default value.
// Used by the frontend.
interface StringPayload {
  type: "STRING";
  default: string;
  value?: string;
}

// StringArrayPayload is the string array type payload configuration options and default value.
// Used by the frontend.
interface StringArrayPayload {
  type: "STRING_ARRAY";
  default: string[];
  value?: string[];
}

// TemplatePayload is the string template type payload configuration options and default value.
// Used by the frontend.
interface TemplatePayload {
  type: "TEMPLATE";
  default: string;
  templateList: { id: string; description?: string }[];
  value?: string;
}

// RuleConfigComponent is the rule configuration options and default value.
// Used by the frontend.
export interface RuleConfigComponent {
  title: string;
  description: string;
  payload: StringPayload | TemplatePayload | StringArrayPayload;
}

// The identifier for rule template
export type RuleType =
  | "engine.mysql.use-innodb"
  | "table.require-pk"
  | "naming.table"
  | "naming.column"
  | "naming.index.pk"
  | "naming.index.uk"
  | "naming.index.fk"
  | "naming.index.idx"
  | "column.required"
  | "column.no-null"
  | "statement.select.no-select-all"
  | "statement.where.require"
  | "statement.where.no-leading-wildcard-like"
  | "schema.backward-compatibility";

// The naming format rule payload.
// Used by the backend.
interface NamingFormatPayload {
  format: string;
}

// The naming format rule payload.
// Used by the backend.
interface RequiredColumnPayload {
  columnList: string[];
}

// The SchemaPolicyRule stores the rule configuration by users.
// Used by the backend
export interface SchemaPolicyRule {
  type: RuleType;
  level: RuleLevel;
  payload?: NamingFormatPayload | RequiredColumnPayload;
}

// The API for schema review policy in backend.
export interface DatabaseSchemaReviewPolicy {
  id: PolicyId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  ruleList: SchemaPolicyRule[];
  environment: Environment;
}

// RuleTemplate is the rule template. Used by the frontend
export interface RuleTemplate {
  type: RuleType;
  category: CategoryType;
  engine: SchemaRuleEngineType;
  componentList: RuleConfigComponent[];
  level: RuleLevel;
}

// SchemaReviewPolicyTemplate is the rule template set
export interface SchemaReviewPolicyTemplate {
  title: string;
  imagePath: string;
  ruleList: RuleTemplate[];
}

// RULE_TEMPLATE_PAYLOAD_MAP is the relationship mapping for the rule type and payload.
// Used by frontend to get different rule payload configurations.
export const RULE_TEMPLATE_PAYLOAD_MAP: Map<RuleType, RuleConfigComponent[]> =
  new Map([
    [
      "naming.table",
      [
        {
          title: "table-name-format",
          description: "",
          payload: {
            type: "STRING",
            default: "^[a-z]+(_[a-z]+)?$",
          },
        },
      ],
    ],
    [
      "naming.column",
      [
        {
          title: "column-name-format",
          description: "",
          payload: {
            type: "STRING",
            default: "^[a-z]+(_[a-z]+)?$",
          },
        },
      ],
    ],
    [
      "naming.index.pk",
      [
        {
          title: "pk-name-format",
          description: "",
          payload: {
            type: "TEMPLATE",
            default: "pk_{{table}}_{{column_list}}",
            templateList: [
              {
                id: "table",
                description:
                  "schema-review-policy.payload-config.template.table-name",
              },
              {
                id: "column_list",
                description:
                  "schema-review-policy.payload-config.template.column-list",
              },
            ],
          },
        },
      ],
    ],
    [
      "naming.index.uk",
      [
        {
          title: "uk-name-format",
          description: "",
          payload: {
            type: "TEMPLATE",
            default: "uk_{{table}}_{{column_list}}",
            templateList: [
              {
                id: "table",
                description:
                  "schema-review-policy.payload-config.template.table-name",
              },
              {
                id: "column_list",
                description:
                  "schema-review-policy.payload-config.template.column-list",
              },
            ],
          },
        },
      ],
    ],
    [
      "naming.index.fk",
      [
        {
          title: "fk-name-format",
          description: "",
          payload: {
            type: "TEMPLATE",
            default:
              "fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}",
            templateList: [
              {
                id: "referencing_table",
                description:
                  "schema-review-policy.payload-config.template.referencing-table",
              },
              {
                id: "referencing_column",
                description:
                  "schema-review-policy.payload-config.template.referencing-column",
              },
              {
                id: "referenced_table",
                description:
                  "schema-review-policy.payload-config.template.referenced-table",
              },
              {
                id: "referenced_column",
                description:
                  "schema-review-policy.payload-config.template.referenced-column",
              },
            ],
          },
        },
      ],
    ],
    [
      "naming.index.idx",
      [
        {
          title: "idx-name-format",
          description: "",
          payload: {
            type: "TEMPLATE",
            default: "idx_{{table}}_{{column_list}}",
            templateList: [
              {
                id: "table",
                description:
                  "schema-review-policy.payload-config.template.table-name",
              },
              {
                id: "column_list",
                description:
                  "schema-review-policy.payload-config.template.column-list",
              },
            ],
          },
        },
      ],
    ],
    [
      "column.required",
      [
        {
          title: "required-column",
          description: "",
          payload: {
            type: "STRING_ARRAY",
            default: [
              "id",
              "created_ts",
              "updated_ts",
              "creator_id",
              "updater_id",
            ],
          },
        },
      ],
    ],
  ]);

// ruleTemplateList stores the default value for each rule template
export const ruleTemplateList: RuleTemplate[] = [
  {
    type: "engine.mysql.use-innodb",
    category: "ENGINE",
    engine: "MYSQL",
    level: RuleLevel.ERROR,
    componentList: [],
  },
  {
    type: "table.require-pk",
    category: "TABLE",
    engine: "COMMON",
    level: RuleLevel.ERROR,
    componentList: [],
  },
  {
    type: "naming.table",
    category: "NAMING",
    engine: "COMMON",
    componentList: RULE_TEMPLATE_PAYLOAD_MAP.get("naming.table") ?? [],
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.column",
    category: "NAMING",
    engine: "COMMON",
    componentList: RULE_TEMPLATE_PAYLOAD_MAP.get("naming.column") ?? [],
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.index.pk",
    category: "NAMING",
    engine: "COMMON",
    componentList: RULE_TEMPLATE_PAYLOAD_MAP.get("naming.index.pk") ?? [],
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.index.uk",
    category: "NAMING",
    engine: "COMMON",
    componentList: RULE_TEMPLATE_PAYLOAD_MAP.get("naming.index.uk") ?? [],
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.index.fk",
    category: "NAMING",
    engine: "COMMON",
    componentList: RULE_TEMPLATE_PAYLOAD_MAP.get("naming.index.fk") ?? [],
    level: RuleLevel.ERROR,
  },
  {
    type: "naming.index.idx",
    category: "NAMING",
    engine: "COMMON",
    componentList: RULE_TEMPLATE_PAYLOAD_MAP.get("naming.index.idx") ?? [],
    level: RuleLevel.ERROR,
  },
  {
    type: "column.required",
    category: "COLUMN",
    engine: "COMMON",
    componentList: RULE_TEMPLATE_PAYLOAD_MAP.get("column.required") ?? [],
    level: RuleLevel.ERROR,
  },
  {
    type: "column.no-null",
    category: "COLUMN",
    engine: "COMMON",
    level: RuleLevel.ERROR,
    componentList: [],
  },
  {
    type: "statement.select.no-select-all",
    category: "STATEMENT",
    engine: "COMMON",
    level: RuleLevel.ERROR,
    componentList: [],
  },
  {
    type: "statement.where.require",
    category: "STATEMENT",
    engine: "COMMON",
    level: RuleLevel.ERROR,
    componentList: [],
  },
  {
    type: "statement.where.no-leading-wildcard-like",
    category: "STATEMENT",
    engine: "COMMON",
    level: RuleLevel.ERROR,
    componentList: [],
  },
  {
    type: "schema.backward-compatibility",
    category: "SCHEMA",
    engine: "MYSQL",
    level: RuleLevel.ERROR,
    componentList: [],
  },
];

interface RuleCategory {
  id: CategoryType;
  ruleList: RuleTemplate[];
}

// convertToCategoryList will reduce RuleTemplate list to RuleCategory list.
export const convertToCategoryList = (
  ruleList: RuleTemplate[]
): RuleCategory[] => {
  const categoryOrder: Map<CategoryType, number> = new Map([
    ["ENGINE", 6],
    ["NAMING", 5],
    ["STATEMENT", 4],
    ["TABLE", 3],
    ["SCHEMA", 2],
    ["COLUMN", 1],
  ]);

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

// The convertRuleTemplateToPolicyRule will convert the review policy rule to rule template for frontend useage.
// Will throw exception if we don't implement the payload handler for specific type of rule.
export const convertPolicyRuleToRuleTemplate = (
  policyRule: SchemaPolicyRule,
  ruleTemplate: RuleTemplate
): RuleTemplate => {
  if (policyRule.type !== ruleTemplate.type) {
    throw new Error(
      `The rule type is not same. policyRule:${ruleTemplate.type}, ruleTemplate:${ruleTemplate.type}`
    );
  }

  const res = { ...ruleTemplate, level: policyRule.level };

  if (ruleTemplate.componentList.length === 0) {
    return res;
  }

  switch (ruleTemplate.type) {
    case "naming.column":
    case "naming.table":
      const stringComponent = ruleTemplate.componentList[0];
      const namingRulepayload = {
        ...stringComponent.payload,
        value: (policyRule.payload as NamingFormatPayload).format,
      } as StringPayload;
      return {
        ...res,
        componentList: [
          {
            ...stringComponent,
            payload: namingRulepayload,
          },
        ],
      };
    case "naming.index.idx":
    case "naming.index.pk":
    case "naming.index.uk":
    case "naming.index.fk":
      const templateComponent = ruleTemplate.componentList[0];
      const indexRulePayload = {
        ...templateComponent.payload,
        value: (policyRule.payload as NamingFormatPayload).format,
      } as TemplatePayload;
      return {
        ...res,
        componentList: [
          {
            ...templateComponent,
            payload: indexRulePayload,
          },
        ],
      };
    case "column.required":
      const requiredColumnComponent = ruleTemplate.componentList[0];
      const requiredColumnPayload = {
        ...requiredColumnComponent.payload,
        value: (policyRule.payload as RequiredColumnPayload).columnList,
      } as StringArrayPayload;
      return {
        ...res,
        componentList: [
          {
            ...requiredColumnComponent,
            payload: requiredColumnPayload,
          },
        ],
      };
  }

  throw new Error(`Invalid rule ${ruleTemplate.type}`);
};

// The convertRuleTemplateToPolicyRule will convert rule template to review policy rule for backend useage.
// Will throw exception if we don't implement the payload handler for specific type of rule.
export const convertRuleTemplateToPolicyRule = (
  rule: RuleTemplate
): SchemaPolicyRule => {
  const base: SchemaPolicyRule = {
    type: rule.type,
    level: rule.level,
  };
  if (rule.componentList.length === 0) {
    return base;
  }

  switch (rule.type) {
    case "naming.column":
    case "naming.table":
      const stringPayload = rule.componentList[0].payload as StringPayload;
      return {
        ...base,
        payload: {
          format: stringPayload.value ?? stringPayload.default,
        },
      };
    case "naming.index.idx":
    case "naming.index.pk":
    case "naming.index.uk":
    case "naming.index.fk":
      const templatePayload = rule.componentList[0].payload as TemplatePayload;
      return {
        ...base,
        payload: {
          format: templatePayload.value ?? templatePayload.default,
        },
      };
    case "column.required":
      const stringArrayPayload = rule.componentList[0]
        .payload as StringArrayPayload;
      return {
        ...base,
        payload: {
          columnList: stringArrayPayload.value ?? stringArrayPayload.default,
        },
      };
  }

  throw new Error(`Invalid rule ${rule.type}`);
};

export const getRuleLocalization = (
  type: RuleType
): { title: string; description: string } => {
  const { t } = useI18n();
  const key = type.split(".").join("-");

  const title = t(`schema-review-policy.rule.${key}.title`);
  const description = t(`schema-review-policy.rule.${key}.description`);

  return {
    title,
    description,
  };
};
