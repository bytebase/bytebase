import { useI18n } from "vue-i18n";
import { PolicyId } from "./id";
import { RowStatus } from "./common";
import { Environment } from "./environment";
import { PlanType } from "./plan";
import sqlReviewSchema from "./sql-review-schema.yaml";
import sqlReviewProdTemplate from "./sql-review.prod.yaml";
import sqlReviewDevTemplate from "./sql-review.dev.yaml";

// The engine type for rule template
export type SchemaRuleEngineType = "MYSQL" | "POSTGRES" | "TIDB";

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
interface NumberPayload {
  type: "NUMBER";
  default: number;
  value?: number;
}

// StringPayload is the string type payload configuration options and default value.
// Used by the frontend.
interface StringPayload {
  type: "STRING";
  default: string;
  value?: string;
}

// BooleanPayload is the boolean type payload configuration options and default value.
// Used by the frontend.
interface BooleanPayload {
  type: "BOOLEAN";
  default: boolean;
  value?: boolean;
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
  | "mysql.engine.mysql.use-innodb"
  | "mysql.naming.table"
  | "mysql.naming.column"
  | "mysql.naming.index.uk"
  | "mysql.naming.index.fk"
  | "mysql.naming.index.idx"
  | "mysql.naming.column.auto-increment"
  | "mysql.statement.select.no-select-all"
  | "mysql.statement.where.require"
  | "mysql.statement.where.no-leading-wildcard-like"
  | "mysql.statement.disallow-commit"
  | "mysql.statement.disallow-limit"
  | "mysql.statement.disallow-order-by"
  | "mysql.statement.merge-alter-table"
  | "mysql.statement.insert.row-limit"
  | "mysql.statement.insert.must-specify-column"
  | "mysql.statement.insert.disallow-order-by-rand"
  | "mysql.statement.affected-row-limit"
  | "mysql.statement.dml-dry-run"
  | "mysql.table.require-pk"
  | "mysql.table.no-foreign-key"
  | "mysql.table.drop-naming-convention"
  | "mysql.table.comment"
  | "mysql.table.disallow-partition"
  | "mysql.column.required"
  | "mysql.column.no-null"
  | "mysql.column.disallow-change-type"
  | "mysql.column.set-default-for-not-null"
  | "mysql.column.disallow-change"
  | "mysql.column.disallow-changing-order"
  | "mysql.column.comment"
  | "mysql.column.auto-increment-must-integer"
  | "mysql.column.type-disallow-list"
  | "mysql.column.disallow-set-charset"
  | "mysql.column.maximum-character-length"
  | "mysql.column.auto-increment-initial-value"
  | "mysql.column.auto-increment-must-unsigned"
  | "mysql.column.current-time-count-limit"
  | "mysql.column.require-default"
  | "mysql.schema.backward-compatibility"
  | "mysql.database.drop-empty-database"
  | "mysql.index.no-duplicate-column"
  | "mysql.index.key-number-limit"
  | "mysql.index.pk-type-limit"
  | "mysql.index.type-no-blob"
  | "mysql.index.total-number-limit"
  | "mysql.system.charset.allowlist"
  | "mysql.system.collation.allowlist"
  | "tidb.naming.table"
  | "tidb.naming.column"
  | "tidb.naming.index.uk"
  | "tidb.naming.index.fk"
  | "tidb.naming.index.idx"
  | "tidb.naming.column.auto-increment"
  | "tidb.statement.select.no-select-all"
  | "tidb.statement.where.require"
  | "tidb.statement.where.no-leading-wildcard-like"
  | "tidb.statement.disallow-commit"
  | "tidb.statement.disallow-limit"
  | "tidb.statement.disallow-order-by"
  | "tidb.statement.merge-alter-table"
  | "tidb.statement.insert.must-specify-column"
  | "tidb.statement.insert.disallow-order-by-rand"
  | "tidb.statement.dml-dry-run"
  | "tidb.table.require-pk"
  | "tidb.table.no-foreign-key"
  | "tidb.table.drop-naming-convention"
  | "tidb.table.comment"
  | "tidb.table.disallow-partition"
  | "tidb.column.required"
  | "tidb.column.no-null"
  | "tidb.column.disallow-change-type"
  | "tidb.column.set-default-for-not-null"
  | "tidb.column.disallow-change"
  | "tidb.column.disallow-changing-order"
  | "tidb.column.comment"
  | "tidb.column.auto-increment-must-integer"
  | "tidb.column.type-disallow-list"
  | "tidb.column.disallow-set-charset"
  | "tidb.column.maximum-character-length"
  | "tidb.column.auto-increment-initial-value"
  | "tidb.column.auto-increment-must-unsigned"
  | "tidb.column.current-time-count-limit"
  | "tidb.column.require-default"
  | "tidb.schema.backward-compatibility"
  | "tidb.database.drop-empty-database"
  | "tidb.index.no-duplicate-column"
  | "tidb.index.key-number-limit"
  | "tidb.index.pk-type-limit"
  | "tidb.index.type-no-blob"
  | "tidb.index.total-number-limit"
  | "tidb.system.charset.allowlist"
  | "tidb.system.collation.allowlist"
  | "pg.naming.table"
  | "pg.naming.column"
  | "pg.naming.index.pk"
  | "pg.naming.index.uk"
  | "pg.naming.index.fk"
  | "pg.naming.index.idx"
  | "pg.statement.select.no-select-all"
  | "pg.statement.where.require"
  | "pg.statement.where.no-leading-wildcard-like"
  | "pg.statement.disallow-commit"
  | "pg.statement.merge-alter-table"
  | "pg.statement.insert.row-limit"
  | "pg.statement.insert.must-specify-column"
  | "pg.statement.insert.disallow-order-by-rand"
  | "pg.statement.affected-row-limit"
  | "pg.statement.dml-dry-run"
  | "pg.statement.disallow-add-column-with-default"
  | "pg.statement.add-check-not-valid"
  | "pg.statement.disallow-add-not-null"
  | "pg.table.require-pk"
  | "pg.table.no-foreign-key"
  | "pg.table.drop-naming-convention"
  | "pg.table.disallow-partition"
  | "pg.column.required"
  | "pg.column.no-null"
  | "pg.column.disallow-change-type"
  | "pg.column.type-disallow-list"
  | "pg.column.maximum-character-length"
  | "pg.column.require-default"
  | "pg.schema.backward-compatibility"
  | "pg.index.no-duplicate-column"
  | "pg.index.key-number-limit"
  | "pg.index.total-number-limit"
  | "pg.index.primary-key-type-allowlist"
  | "pg.index.create-concurrently"
  | "pg.system.charset.allowlist"
  | "pg.system.collation.allowlist"
  | "pg.comment.length";

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

// The SchemaPolicyRule stores the rule configuration by users.
// Used by the backend
export interface SchemaPolicyRule {
  type: RuleType;
  level: RuleLevel;
  payload?:
    | NamingFormatPayload
    | StringArrayLimitPayload
    | CommentFormatPayload
    | NumberLimitPayload;
}

// The API for SQL review policy in backend.
export interface SQLReviewPolicy {
  id: PolicyId;

  // Standard fields
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
  engineList: SchemaRuleEngineType[];
  componentList: RuleConfigComponent[];
  level: RuleLevel;
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
  const templateList = [sqlReviewProdTemplate, sqlReviewDevTemplate] as {
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

  const res = { ...ruleTemplate, level: policyRule.level };

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
    case "mysql.table.drop-naming-convention":
    case "tidb.table.drop-naming-convention":
    case "pg.table.drop-naming-convention":
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
    case "mysql.naming.column":
    case "mysql.naming.column.auto-increment":
    case "mysql.naming.table":
    case "tidb.naming.column":
    case "tidb.naming.column.auto-increment":
    case "tidb.naming.table":
    case "pg.naming.column":
    case "pg.naming.table":
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
    case "pg.naming.index.pk":
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
    case "mysql.naming.index.idx":
    case "mysql.naming.index.uk":
    case "mysql.naming.index.fk":
    case "tidb.naming.index.idx":
    case "tidb.naming.index.uk":
    case "tidb.naming.index.fk":
    case "pg.naming.index.idx":
    case "pg.naming.index.uk":
    case "pg.naming.index.fk":
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
    case "mysql.column.required":
    case "tidb.column.required":
    case "pg.column.required": {
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
    case "mysql.column.type-disallow-list":
    case "mysql.system.charset.allowlist":
    case "mysql.system.collation.allowlist":
    case "tidb.column.type-disallow-list":
    case "tidb.system.charset.allowlist":
    case "tidb.system.collation.allowlist":
    case "pg.column.type-disallow-list":
    case "pg.system.charset.allowlist":
    case "pg.system.collation.allowlist": {
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
    case "mysql.column.comment":
    case "mysql.table.comment":
    case "tidb.column.comment":
    case "tidb.table.comment":
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
    case "mysql.statement.insert.row-limit":
    case "mysql.statement.affected-row-limit":
    case "mysql.column.maximum-character-length":
    case "mysql.column.auto-increment-initial-value":
    case "mysql.index.key-number-limit":
    case "mysql.index.total-number-limit":
    case "tidb.column.maximum-character-length":
    case "tidb.column.auto-increment-initial-value":
    case "tidb.index.key-number-limit":
    case "tidb.index.total-number-limit":
    case "pg.statement.insert.row-limit":
    case "pg.statement.affected-row-limit":
    case "pg.column.maximum-character-length":
    case "pg.index.key-number-limit":
    case "pg.index.total-number-limit":
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
    case "mysql.table.drop-naming-convention":
    case "tidb.table.drop-naming-convention":
    case "pg.table.drop-naming-convention":
      if (!stringPayload) {
        throw new Error(`Invalid rule ${rule.type}`);
      }

      return {
        ...base,
        payload: {
          format: stringPayload.value ?? stringPayload.default,
        },
      };
    case "mysql.naming.column":
    case "mysql.naming.column.auto-increment":
    case "mysql.naming.table":
    case "tidb.naming.column":
    case "tidb.naming.column.auto-increment":
    case "tidb.naming.table":
    case "pg.naming.column":
    case "pg.naming.table":
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
    case "pg.naming.index.pk":
      if (!templatePayload) {
        throw new Error(`Invalid rule ${rule.type}`);
      }
      return {
        ...base,
        payload: {
          format: templatePayload.value ?? templatePayload.default,
        },
      };
    case "mysql.naming.index.idx":
    case "mysql.naming.index.uk":
    case "mysql.naming.index.fk":
    case "tidb.naming.index.idx":
    case "tidb.naming.index.uk":
    case "tidb.naming.index.fk":
    case "pg.naming.index.idx":
    case "pg.naming.index.uk":
    case "pg.naming.index.fk":
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
    case "mysql.column.required":
    case "mysql.column.type-disallow-list":
    case "mysql.system.charset.allowlist":
    case "mysql.system.collation.allowlist":
    case "tidb.column.required":
    case "tidb.column.type-disallow-list":
    case "tidb.system.charset.allowlist":
    case "tidb.system.collation.allowlist":
    case "pg.column.required":
    case "pg.column.type-disallow-list":
    case "pg.system.charset.allowlist":
    case "pg.system.collation.allowlist": {
      const stringArrayPayload = rule.componentList[0]
        .payload as StringArrayPayload;
      return {
        ...base,
        payload: {
          list: stringArrayPayload.value ?? stringArrayPayload.default,
        },
      };
    }
    case "mysql.column.comment":
    case "mysql.table.comment":
    case "tidb.column.comment":
    case "tidb.table.comment":
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
    case "mysql.statement.insert.row-limit":
    case "mysql.statement.affected-row-limit":
    case "mysql.column.maximum-character-length":
    case "mysql.column.auto-increment-initial-value":
    case "mysql.index.key-number-limit":
    case "mysql.index.total-number-limit":
    case "tidb.column.maximum-character-length":
    case "tidb.column.auto-increment-initial-value":
    case "tidb.index.key-number-limit":
    case "tidb.index.total-number-limit":
    case "pg.statement.insert.row-limit":
    case "pg.statement.affected-row-limit":
    case "pg.column.maximum-character-length":
    case "pg.index.key-number-limit":
    case "pg.index.total-number-limit":
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

const availableRulesForFreePlan: RuleType[] = [
  "mysql.statement.where.require",
  "mysql.column.no-null",
  "tidb.statement.where.require",
  "tidb.column.no-null",
  "pg.statement.where.require",
  "pg.column.no-null",
];

export const ruleIsAvailableInSubscription = (
  ruleType: RuleType,
  subscriptionPlan: PlanType
): boolean => {
  if (subscriptionPlan === PlanType.FREE) {
    return availableRulesForFreePlan.indexOf(ruleType) >= 0;
  }
  return true;
};
