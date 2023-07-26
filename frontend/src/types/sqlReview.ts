import { useI18n } from "vue-i18n";
import { PolicyId } from "./id";
import { Environment } from "@/types/proto/v1/environment_service";
import { PlanType } from "@/types/proto/v1/subscription_service";
import sqlReviewSchema from "./sql-review-schema.yaml";
import sqlReviewSampleTemplate from "./sql-review.sample.yaml";
import sqlReviewProdTemplate from "./sql-review.prod.yaml";
import sqlReviewDevTemplate from "./sql-review.dev.yaml";

// The engine type for rule template
export type SchemaRuleEngineType =
  | "MYSQL"
  | "POSTGRES"
  | "TIDB"
  | "ORACLE"
  | "DM"
  | "OCEANBASE"
  | "SNOWFLKE"
  | "MSSQL";

// The category type for rule template
export type CategoryType =
  | "ENGINE"
  | "NAMING"
  | "STATEMENT"
  | "TABLE"
  | "COLUMN"
  | "SCHEMA"
  | "DATABASE"
  | "INDEX"
  | "SYSTEM";

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

// NumberPayload is the number type payload configuration options and default value.
// Used by the frontend.
export interface NumberPayload {
  type: "NUMBER";
  default: number;
  value?: number;
}

// StringPayload is the string type payload configuration options and default value.
// Used by the frontend.
export interface StringPayload {
  type: "STRING";
  default: string;
  value?: string;
}

// BooleanPayload is the boolean type payload configuration options and default value.
// Used by the frontend.
export interface BooleanPayload {
  type: "BOOLEAN";
  default: boolean;
  value?: boolean;
}

// StringArrayPayload is the string array type payload configuration options and default value.
// Used by the frontend.
export interface StringArrayPayload {
  type: "STRING_ARRAY";
  default: string[];
  value?: string[];
}

// TemplatePayload is the string template type payload configuration options and default value.
// Used by the frontend.
export interface TemplatePayload {
  type: "TEMPLATE";
  default: string;
  templateList: string[];
  value?: string;
}

// RuleConfigComponent is the rule configuration options and default value.
// Used by the frontend.
export interface RuleConfigComponent {
  key: string;
  payload:
    | StringPayload
    | NumberPayload
    | TemplatePayload
    | StringArrayPayload
    | BooleanPayload;
}

// The identifier for rule template
export type RuleType =
  | "engine.mysql.use-innodb"
  | "table.require-pk"
  | "table.no-foreign-key"
  | "table.drop-naming-convention"
  | "table.disallow-partition"
  | "table.comment"
  | "naming.table"
  | "naming.column"
  | "naming.index.uk"
  | "naming.index.pk"
  | "naming.index.fk"
  | "naming.index.idx"
  | "naming.column.auto-increment"
  | "naming.table.no-keyword"
  | "naming.identifier.no-keyword"
  | "naming.identifier.case"
  | "column.required"
  | "column.no-null"
  | "column.comment"
  | "column.type-disallow-list"
  | "column.disallow-change-type"
  | "column.set-default-for-not-null"
  | "column.disallow-change"
  | "column.disallow-changing-order"
  | "column.auto-increment-must-integer"
  | "column.disallow-set-charset"
  | "column.auto-increment-must-unsigned"
  | "column.maximum-character-length"
  | "column.maximum-varchar-length"
  | "column.auto-increment-initial-value"
  | "column.current-time-count-limit"
  | "column.require-default"
  | "statement.select.no-select-all"
  | "statement.where.require"
  | "statement.where.no-leading-wildcard-like"
  | "statement.disallow-commit"
  | "statement.disallow-limit"
  | "statement.disallow-order-by"
  | "statement.merge-alter-table"
  | "statement.insert.must-specify-column"
  | "statement.insert.disallow-order-by-rand"
  | "statement.insert.row-limit"
  | "statement.affected-row-limit"
  | "statement.dml-dry-run"
  | "statement.disallow-add-column-with-default"
  | "statement.add-check-not-valid"
  | "statement.disallow-add-not-null"
  | "schema.backward-compatibility"
  | "database.drop-empty-database"
  | "system.charset.allowlist"
  | "system.collation.allowlist"
  | "system.comment.length"
  | "index.no-duplicate-column"
  | "index.type-no-blob"
  | "index.key-number-limit"
  | "index.total-number-limit"
  | "index.primary-key-type-allowlist"
  | "index.create-concurrently"
  | "index.pk-type-limit";

// The naming format rule payload.
// Used by the backend.
interface NamingFormatPayload {
  format: string;
  maxLength?: number;
}

// The string array rule payload.
// Used by the backend.
interface StringArrayLimitPayload {
  list: string[];
}

// The comment format rule payload.
// Used by the backend.
interface CommentFormatPayload {
  required: boolean;
  maxLength: number;
}

// The number limit rule payload.
// Used by the backend.
interface NumberLimitPayload {
  number: number;
}

// The case rule payload.
// Used by the backend.
interface CasePayload {
  upper: boolean;
}

// The SchemaPolicyRule stores the rule configuration by users.
// Used by the backend
export interface SchemaPolicyRule {
  type: RuleType;
  level: RuleLevel;
  payload?:
    | NamingFormatPayload
    | StringArrayLimitPayload
    | CommentFormatPayload
    | NumberLimitPayload
    | CasePayload;
  comment: string;
}

// The API for SQL review policy in backend.
export interface SQLReviewPolicy {
  id: PolicyId;

  // Standard fields
  // enforce means if the policy is active
  enforce: boolean;

  // Domain specific fields
  name: string;
  ruleList: SchemaPolicyRule[];
  environment: Environment;
}

// RuleTemplate is the rule template. Used by the frontend
export interface RuleTemplate {
  type: RuleType;
  category: CategoryType;
  engineList: SchemaRuleEngineType[];
  componentList: RuleConfigComponent[];
  level: RuleLevel;
  comment?: string;
}

// SQLReviewPolicyTemplate is the rule template set
export interface SQLReviewPolicyTemplate {
  id: string;
  ruleList: RuleTemplate[];
}

// Build the frontend template list based on schema and template.
export const TEMPLATE_LIST: SQLReviewPolicyTemplate[] = (function () {
  const ruleSchemaMap = (sqlReviewSchema.ruleList as RuleTemplate[]).reduce(
    (map, ruleSchema) => {
      map.set(ruleSchema.type, ruleSchema);
      return map;
    },
    new Map<RuleType, RuleTemplate>()
  );
  const templateList = [
    sqlReviewSampleTemplate,
    sqlReviewDevTemplate,
    sqlReviewProdTemplate,
  ] as {
    id: string;
    ruleList: {
      type: RuleType;
      level: RuleLevel;
      payload?: { [key: string]: any };
    }[];
  }[];

  return templateList.map((template) => {
    const ruleList: RuleTemplate[] = [];

    for (const rule of template.ruleList) {
      const ruleTemplate = ruleSchemaMap.get(rule.type);
      if (!ruleTemplate) {
        continue;
      }

      // Using template rule payload to override the component list.
      const componentList = ruleTemplate.componentList.map((component) => {
        if (rule.payload && rule.payload[component.key]) {
          return {
            ...component,
            payload: {
              ...component.payload,
              default: rule.payload[component.key],
            },
          };
        }
        return component;
      });
      ruleList.push({
        ...ruleTemplate,
        level: rule.level,
        componentList,
      });
    }

    return {
      id: template.id,
      ruleList,
    };
  });
})();

export const findRuleTemplate = (type: RuleType) => {
  for (let i = 0; i < TEMPLATE_LIST.length; i++) {
    const template = TEMPLATE_LIST[i];
    const rule = template.ruleList.find((rule) => rule.type === type);
    if (rule) return rule;
  }
  return undefined;
};

export const ruleTemplateMap: Map<RuleType, RuleTemplate> =
  TEMPLATE_LIST.reduce((map, template) => {
    for (const rule of template.ruleList) {
      map.set(rule.type, rule);
    }
    return map;
  }, new Map<RuleType, RuleTemplate>());

interface RuleCategory {
  id: CategoryType;
  ruleList: RuleTemplate[];
}

// convertToCategoryList will reduce RuleTemplate list to RuleCategory list.
export const convertToCategoryList = (
  ruleList: RuleTemplate[]
): RuleCategory[] => {
  const categoryList = sqlReviewSchema.categoryList as CategoryType[];
  const categoryOrder = categoryList.reduce((map, category, index) => {
    map.set(category, index);
    return map;
  }, new Map<CategoryType, number>());

  const dict = ruleList.reduce((dict, rule) => {
    if (!dict[rule.category]) {
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

  const res = {
    ...ruleTemplate,
    level: policyRule.level,
    comment: policyRule.comment,
  };

  if (ruleTemplate.componentList.length === 0) {
    return res;
  }

  const stringComponent = ruleTemplate.componentList.find(
    (c) => c.payload.type === "STRING"
  );
  const numberComponent = ruleTemplate.componentList.find(
    (c) => c.payload.type === "NUMBER"
  );
  const booleanComponent = ruleTemplate.componentList.find(
    (c) => c.payload.type === "BOOLEAN"
  );
  const templateComponent = ruleTemplate.componentList.find(
    (c) => c.payload.type === "TEMPLATE"
  );

  switch (ruleTemplate.type) {
    case "table.drop-naming-convention":
      if (!stringComponent) {
        throw new Error(`Invalid rule ${ruleTemplate.type}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...stringComponent,
            payload: {
              ...stringComponent.payload,
              value: (policyRule.payload as NamingFormatPayload).format,
            } as StringPayload,
          },
        ],
      };
    case "naming.column":
    case "naming.column.auto-increment":
    case "naming.table":
      if (!stringComponent || !numberComponent) {
        throw new Error(`Invalid rule ${ruleTemplate.type}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...stringComponent,
            payload: {
              ...stringComponent.payload,
              value: (policyRule.payload as NamingFormatPayload).format,
            } as StringPayload,
          },
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: (policyRule.payload as NamingFormatPayload).maxLength,
            } as NumberPayload,
          },
        ],
      };
    case "naming.index.pk":
      if (!templateComponent) {
        throw new Error(`Invalid rule ${ruleTemplate.type}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...templateComponent,
            payload: {
              ...templateComponent.payload,
              value: (policyRule.payload as NamingFormatPayload).format,
            } as TemplatePayload,
          },
        ],
      };
    case "naming.index.idx":
    case "naming.index.uk":
    case "naming.index.fk":
      if (!templateComponent || !numberComponent) {
        throw new Error(`Invalid rule ${ruleTemplate.type}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...templateComponent,
            payload: {
              ...templateComponent.payload,
              value: (policyRule.payload as NamingFormatPayload).format,
            } as TemplatePayload,
          },
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: (policyRule.payload as NamingFormatPayload).maxLength,
            } as NumberPayload,
          },
        ],
      };
    case "naming.identifier.case":
      if (!booleanComponent) {
        throw new Error(`Invalid rule ${ruleTemplate.type}`);
      }
      return {
        ...res,
        componentList: [
          {
            ...booleanComponent,
            payload: booleanComponent.payload,
          },
        ],
      };
    case "column.required": {
      const requiredColumnComponent = ruleTemplate.componentList[0];
      // The columnList payload is deprecated.
      // Just keep it to compatible with old data, we can remove this later.
      let value: string[] = (policyRule.payload as any)["columnList"];
      if (!value) {
        value = (policyRule.payload as StringArrayLimitPayload).list;
      }
      const requiredColumnPayload = {
        ...requiredColumnComponent.payload,
        value,
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
    case "column.type-disallow-list":
    case "index.primary-key-type-allowlist":
    case "system.charset.allowlist":
    case "system.collation.allowlist": {
      const stringArrayComponent = ruleTemplate.componentList[0];
      const stringArrayPayload = {
        ...stringArrayComponent.payload,
        value: (policyRule.payload as StringArrayLimitPayload).list,
      } as StringArrayPayload;
      return {
        ...res,
        componentList: [
          {
            ...stringArrayComponent,
            payload: stringArrayPayload,
          },
        ],
      };
    }
    case "column.comment":
    case "table.comment":
      if (!booleanComponent || !numberComponent) {
        throw new Error(`Invalid rule ${ruleTemplate.type}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...booleanComponent,
            payload: {
              ...booleanComponent.payload,
              value: (policyRule.payload as CommentFormatPayload).required,
            } as BooleanPayload,
          },
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: (policyRule.payload as CommentFormatPayload).maxLength,
            } as NumberPayload,
          },
        ],
      };
    case "statement.insert.row-limit":
    case "statement.affected-row-limit":
    case "column.maximum-character-length":
    case "column.maximum-varchar-length":
    case "column.auto-increment-initial-value":
    case "index.key-number-limit":
    case "index.total-number-limit":
    case "system.comment.length":
      if (!numberComponent) {
        throw new Error(`Invalid rule ${ruleTemplate.type}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: (policyRule.payload as NumberLimitPayload).number,
            } as NumberPayload,
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
    comment: rule.comment ?? "",
  };
  if (rule.componentList.length === 0) {
    return base;
  }

  const stringPayload = rule.componentList.find(
    (c) => c.payload.type === "STRING"
  )?.payload as StringPayload | undefined;
  const numberPayload = rule.componentList.find(
    (c) => c.payload.type === "NUMBER"
  )?.payload as NumberPayload | undefined;
  const templatePayload = rule.componentList.find(
    (c) => c.payload.type === "TEMPLATE"
  )?.payload as TemplatePayload | undefined;
  const booleanPayload = rule.componentList.find(
    (c) => c.payload.type === "BOOLEAN"
  )?.payload as BooleanPayload | undefined;

  switch (rule.type) {
    case "table.drop-naming-convention":
      if (!stringPayload) {
        throw new Error(`Invalid rule ${rule.type}`);
      }

      return {
        ...base,
        payload: {
          format: stringPayload.value ?? stringPayload.default,
        },
      };
    case "naming.column":
    case "naming.column.auto-increment":
    case "naming.table":
      if (!stringPayload || !numberPayload) {
        throw new Error(`Invalid rule ${rule.type}`);
      }

      return {
        ...base,
        payload: {
          format: stringPayload.value ?? stringPayload.default,
          maxLength: numberPayload.value ?? numberPayload.default,
        },
      };
    case "naming.index.pk":
      if (!templatePayload) {
        throw new Error(`Invalid rule ${rule.type}`);
      }
      return {
        ...base,
        payload: {
          format: templatePayload.value ?? templatePayload.default,
        },
      };
    case "naming.index.idx":
    case "naming.index.uk":
    case "naming.index.fk":
      if (!templatePayload || !numberPayload) {
        throw new Error(`Invalid rule ${rule.type}`);
      }
      return {
        ...base,
        payload: {
          format: templatePayload.value ?? templatePayload.default,
          maxLength: numberPayload.value ?? numberPayload.default,
        },
      };
    case "naming.identifier.case":
      if (!booleanPayload) {
        throw new Error(`Invalid rule ${rule.type}`);
      }
      return {
        ...base,
        payload: {
          upper: booleanPayload.value ?? booleanPayload.default,
        },
      };
    case "column.required":
    case "column.type-disallow-list":
    case "index.primary-key-type-allowlist":
    case "system.charset.allowlist":
    case "system.collation.allowlist": {
      const stringArrayPayload = rule.componentList[0]
        .payload as StringArrayPayload;
      return {
        ...base,
        payload: {
          list: stringArrayPayload.value ?? stringArrayPayload.default,
        },
      };
    }
    case "column.comment":
    case "table.comment":
      if (!booleanPayload || !numberPayload) {
        throw new Error(`Invalid rule ${rule.type}`);
      }
      return {
        ...base,
        payload: {
          required: booleanPayload.value ?? booleanPayload.default,
          maxLength: numberPayload.value ?? numberPayload.default,
        },
      };
    case "statement.insert.row-limit":
    case "statement.affected-row-limit":
    case "column.maximum-character-length":
    case "column.maximum-varchar-length":
    case "column.auto-increment-initial-value":
    case "index.key-number-limit":
    case "index.total-number-limit":
    case "system.comment.length":
      if (!numberPayload) {
        throw new Error(`Invalid rule ${rule.type}`);
      }
      return {
        ...base,
        payload: {
          number: numberPayload.value ?? numberPayload.default,
        },
      };
  }

  throw new Error(`Invalid rule ${rule.type}`);
};

export const getRuleLocalizationKey = (type: RuleType): string => {
  return type.split(".").join("-");
};

export const getRuleLocalization = (
  type: RuleType
): { title: string; description: string } => {
  const { t } = useI18n();
  const key = getRuleLocalizationKey(type);

  const title = t(`sql-review.rule.${key}.title`);
  const description = t(`sql-review.rule.${key}.description`);

  return {
    title,
    description,
  };
};

export const ruleIsAvailableInSubscription = (
  ruleType: RuleType,
  subscriptionPlan: PlanType
): boolean => {
  return true;
};
