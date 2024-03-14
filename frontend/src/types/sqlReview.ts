import { pullAt, cloneDeep, groupBy } from "lodash-es";
import { useI18n } from "vue-i18n";
import { Engine, engineFromJSON } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";
import {
  SQLReviewRuleLevel,
  sQLReviewRuleLevelFromJSON,
} from "@/types/proto/v1/org_policy_service";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { PolicyId } from "./id";
import sqlReviewSchema from "./sql-review-schema.yaml";
import sqlReviewDevTemplate from "./sql-review.dev.yaml";
import sqlReviewProdTemplate from "./sql-review.prod.yaml";
import sqlReviewSampleTemplate from "./sql-review.sample.yaml";

export const LEVEL_LIST = [
  SQLReviewRuleLevel.ERROR,
  SQLReviewRuleLevel.WARNING,
  SQLReviewRuleLevel.DISABLED,
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

interface IndividualConfigPayload {
  [key: string]: {
    default: any;
    value?: any;
  };
}

export interface IndividualConfigForEngine {
  engine: Engine;
  payload: IndividualConfigPayload;
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

// The string rule payload.
// Used by the backend.
interface StringPayload {
  string: string;
}

// The case rule payload.
// Used by the backend.
interface CasePayload {
  upper: boolean;
}

// The SchemaPolicyRule stores the rule configuration by users.
// Used by the backend
export interface SchemaPolicyRule {
  type: string;
  level: SQLReviewRuleLevel;
  engine: Engine;
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
  type: string;
  category: string;
  engineList: Engine[];
  componentList: RuleConfigComponent[];
  individualConfigList: IndividualConfigForEngine[];
  level: SQLReviewRuleLevel;
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
      map.set(ruleSchema.type, {
        ...ruleSchema,
        componentList: ruleSchema.componentList || [],
        individualConfigList: ruleSchema.individualConfigList || [],
      });
      return map;
    },
    new Map<string, RuleTemplate>()
  );

  interface PayloadObject {
    [key: string]: any;
  }
  const templateList = [
    sqlReviewSampleTemplate,
    sqlReviewDevTemplate,
    sqlReviewProdTemplate,
  ] as {
    id: string;
    ruleList: {
      type: string;
      level: SQLReviewRuleLevel;
      payload?: PayloadObject;
      engine?: Engine;
    }[];
  }[];

  return templateList.map((template) => {
    const ruleList: RuleTemplate[] = [];

    const groupRuleList = groupBy(template.ruleList, (rule) => rule.type);
    for (const [ruleType, groupList] of Object.entries(groupRuleList)) {
      const ruleTemplate = ruleSchemaMap.get(ruleType);
      if (!ruleTemplate) {
        continue;
      }

      const individualConfigList = cloneDeep(
        ruleTemplate.individualConfigList || []
      );
      let componentList = cloneDeep(ruleTemplate.componentList);
      let level = SQLReviewRuleLevel.DISABLED;

      for (const rule of groupList) {
        level = rule.level;
        const index = individualConfigList.findIndex(
          (config) => config.engine === rule.engine
        );
        if (index >= 0) {
          // Override individual config for specific engine.
          individualConfigList[index] = {
            ...individualConfigList[index],
            // Note: it's important to convert string type engine to enum.
            engine: engineFromJSON(individualConfigList[index].engine),
            payload: Object.assign(
              {},
              individualConfigList[index].payload,
              Object.entries(rule.payload ?? {}).reduce((obj, [key, val]) => {
                obj[key] = {
                  default: val,
                };
                return obj;
              }, {} as PayloadObject)
            ),
          };
        } else if (!rule.engine) {
          // Using template rule payload to override the component list.
          componentList = ruleTemplate.componentList.map((component) => {
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
        }
      }

      ruleList.push({
        ...ruleTemplate,
        level: sQLReviewRuleLevelFromJSON(level),
        componentList,
        individualConfigList,
        // Note: it's important to convert string type engine to enum.
        engineList: ruleTemplate.engineList.map((engine) =>
          engineFromJSON(engine)
        ),
      });
    }

    return {
      id: template.id,
      ruleList,
    };
  });
})();

export const findRuleTemplate = (type: string) => {
  for (let i = 0; i < TEMPLATE_LIST.length; i++) {
    const template = TEMPLATE_LIST[i];
    const rule = template.ruleList.find((rule) => rule.type === type);
    if (rule) return rule;
  }
  return undefined;
};

export const ruleTemplateMap: Map<string, RuleTemplate> = TEMPLATE_LIST.reduce(
  (map, template) => {
    for (const rule of template.ruleList) {
      map.set(rule.type, rule);
    }
    return map;
  },
  new Map<string, RuleTemplate>()
);

interface RuleCategory {
  id: string;
  ruleList: RuleTemplate[];
}

// convertToCategoryList will reduce RuleTemplate list to RuleCategory list.
export const convertToCategoryList = (
  ruleList: RuleTemplate[]
): RuleCategory[] => {
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

  return Object.values(dict);
};

const mergeRulePayloadAsIndividualConfig = (
  individualConfig: IndividualConfigForEngine,
  rule: SchemaPolicyRule
): IndividualConfigForEngine => {
  const payload = cloneDeep(individualConfig.payload);
  for (const [key, val] of Object.entries(rule.payload ?? {})) {
    if (!payload[key]) {
      continue;
    }
    payload[key].value = val;
  }
  return {
    ...individualConfig,
    payload,
  };
};

// The convertPolicyRuleToRuleTemplate will convert the review policy rule to rule template for frontend useage.
// Will throw exception if we don't implement the payload handler for specific type of rule.
export const convertPolicyRuleToRuleTemplate = (
  policyRuleList: SchemaPolicyRule[],
  ruleTemplate: RuleTemplate
): RuleTemplate => {
  if (
    policyRuleList.length === 0 ||
    policyRuleList[0].type !== ruleTemplate.type
  ) {
    throw new Error(
      `The rule type is not same. policyRule:${ruleTemplate.type}, ruleTemplate:${ruleTemplate.type}`
    );
  }

  let policyRule: SchemaPolicyRule | undefined = policyRuleList[0];

  const res = {
    ...ruleTemplate,
    level: policyRule.level,
    comment: policyRule.comment,
  };

  const componentList = ruleTemplate.componentList ?? [];
  if (componentList.length === 0) {
    return res;
  }

  const individualConfigList: IndividualConfigForEngine[] = [];
  for (const individualConfig of ruleTemplate.individualConfigList) {
    const index = policyRuleList.findIndex(
      (rule) => rule.engine == individualConfig.engine
    );
    if (index >= 0) {
      const individualRule = policyRuleList[index];
      pullAt(policyRuleList, index);
      individualConfigList.push(
        mergeRulePayloadAsIndividualConfig(individualConfig, individualRule)
      );
    }
  }

  policyRule = policyRuleList[0];
  const payload = policyRule?.payload ?? {};

  const stringComponent = componentList.find(
    (c) => c.payload.type === "STRING"
  );
  const numberComponent = componentList.find(
    (c) => c.payload.type === "NUMBER"
  );
  const booleanComponent = componentList.find(
    (c) => c.payload.type === "BOOLEAN"
  );
  const templateComponent = componentList.find(
    (c) => c.payload.type === "TEMPLATE"
  );

  switch (ruleTemplate.type) {
    case "statement.query.minimum-plan-level":
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
              value: (payload as StringPayload).string,
            } as StringPayload,
          },
        ],
        individualConfigList,
      };
    // Following rules require STRING component.
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
              value: (payload as NamingFormatPayload).format,
            } as StringPayload,
          },
        ],
        individualConfigList,
      };
    // Following rules require STRING and NUMBER component.
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
              value: (payload as NamingFormatPayload).format,
            } as StringPayload,
          },
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: (payload as NamingFormatPayload).maxLength,
            } as NumberPayload,
          },
        ],
        individualConfigList,
      };
    // Following rules require TEMPLATE component.
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
              value: (payload as NamingFormatPayload).format,
            } as TemplatePayload,
          },
        ],
        individualConfigList,
      };
    // Following rules require TEMPLATE and NUMBER component.
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
              value: (payload as NamingFormatPayload).format,
            } as TemplatePayload,
          },
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: (payload as NamingFormatPayload).maxLength,
            } as NumberPayload,
          },
        ],
        individualConfigList,
      };
    // Following rules require BOOLEAN component.
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
        individualConfigList,
      };
    case "column.required": {
      const requiredColumnComponent = componentList[0];
      // The columnList payload is deprecated.
      // Just keep it to compatible with old data, we can remove this later.
      let value: string[] = (payload as any)["columnList"];
      if (!value) {
        value = (payload as StringArrayLimitPayload).list;
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
        individualConfigList,
      };
    }
    // Following rules require STRING_ARRAY component.
    case "column.type-disallow-list":
    case "index.primary-key-type-allowlist":
    case "index.type-allow-list":
    case "system.charset.allowlist":
    case "system.collation.allowlist":
    case "system.function.disallowed-list": {
      const stringArrayComponent = componentList[0];
      const stringArrayPayload = {
        ...stringArrayComponent.payload,
        value: (payload as StringArrayLimitPayload).list,
      } as StringArrayPayload;
      return {
        ...res,
        componentList: [
          {
            ...stringArrayComponent,
            payload: stringArrayPayload,
          },
        ],
        individualConfigList,
      };
    }
    // Following rules require BOOLEAN and NUMBER component.
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
              value: (payload as CommentFormatPayload).required,
            } as BooleanPayload,
          },
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: (payload as CommentFormatPayload).maxLength,
            } as NumberPayload,
          },
        ],
        individualConfigList,
      };
    // Following rules require NUMBER component.
    case "statement.insert.row-limit":
    case "statement.affected-row-limit":
    case "column.maximum-character-length":
    case "column.maximum-varchar-length":
    case "column.auto-increment-initial-value":
    case "index.key-number-limit":
    case "index.total-number-limit":
    case "system.comment.length":
    case "advice.online-migration":
    case "table.text-fields-total-length":
    case "statement.where.maximum-logical-operator-count":
    case "statement.maximum-limit-value":
    case "statement.maximum-join-table-count":
    case "statement.maximum-statements-in-transaction":
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
              value: (payload as NumberLimitPayload).number,
            } as NumberPayload,
          },
        ],
        individualConfigList,
      };
  }

  throw new Error(`Invalid rule ${ruleTemplate.type}`);
};

const mergeIndividualConfigAsRule = (
  base: SchemaPolicyRule,
  template: RuleTemplate
): SchemaPolicyRule => {
  const componentList = template.componentList ?? [];
  const stringPayload = componentList.find((c) => c.payload.type === "STRING")
    ?.payload as StringPayload | undefined;
  const numberPayload = componentList.find((c) => c.payload.type === "NUMBER")
    ?.payload as NumberPayload | undefined;
  const templatePayload = componentList.find(
    (c) => c.payload.type === "TEMPLATE"
  )?.payload as TemplatePayload | undefined;
  const booleanPayload = componentList.find((c) => c.payload.type === "BOOLEAN")
    ?.payload as BooleanPayload | undefined;
  const stringArrayPayload = componentList.find(
    (c) => c.payload.type === "STRING_ARRAY"
  )?.payload as StringArrayPayload | undefined;

  switch (template.type) {
    case "statement.query.minimum-plan-level":
      if (!stringPayload) {
        throw new Error(`Invalid rule ${template.type}`);
      }

      return {
        ...base,
        payload: {
          string: stringPayload.value ?? stringPayload.default,
        },
      };
    // Following rules require STRING component.
    case "table.drop-naming-convention":
      if (!stringPayload) {
        throw new Error(`Invalid rule ${template.type}`);
      }

      return {
        ...base,
        payload: {
          format: stringPayload.value ?? stringPayload.default,
        },
      };
    // Following rules require STRING and NUMBER component.
    case "naming.column":
    case "naming.column.auto-increment":
    case "naming.table":
      if (!stringPayload || !numberPayload) {
        throw new Error(`Invalid rule ${template.type}`);
      }

      return {
        ...base,
        payload: {
          format: stringPayload.value ?? stringPayload.default,
          maxLength: numberPayload.value ?? numberPayload.default,
        },
      };
    // Following rules require TEMPLATE component.
    case "naming.index.pk":
      if (!templatePayload) {
        throw new Error(`Invalid rule ${template.type}`);
      }
      return {
        ...base,
        payload: {
          format: templatePayload.value ?? templatePayload.default,
        },
      };
    // Following rules require TEMPLATE and NUMBER component.
    case "naming.index.idx":
    case "naming.index.uk":
    case "naming.index.fk":
      if (!templatePayload || !numberPayload) {
        throw new Error(`Invalid rule ${template.type}`);
      }
      return {
        ...base,
        payload: {
          format: templatePayload.value ?? templatePayload.default,
          maxLength: numberPayload.value ?? numberPayload.default,
        },
      };
    // Following rules require BOOLEAN component.
    case "naming.identifier.case":
      if (!booleanPayload) {
        throw new Error(`Invalid rule ${template.type}`);
      }
      return {
        ...base,
        payload: {
          upper: booleanPayload.value ?? booleanPayload.default,
        },
      };
    // Following rules require STRING_ARRAY component.
    case "column.required":
    case "column.type-disallow-list":
    case "index.primary-key-type-allowlist":
    case "index.type-allow-list":
    case "system.charset.allowlist":
    case "system.collation.allowlist":
    case "system.function.disallowed-list": {
      if (!stringArrayPayload) {
        throw new Error(`Invalid rule ${template.type}`);
      }
      return {
        ...base,
        payload: {
          list: stringArrayPayload.value ?? stringArrayPayload.default,
        },
      };
    }
    // Following rules require BOOLEAN and NUMBER component.
    case "column.comment":
    case "table.comment":
      if (!booleanPayload || !numberPayload) {
        throw new Error(`Invalid rule ${template.type}`);
      }
      return {
        ...base,
        payload: {
          required: booleanPayload.value ?? booleanPayload.default,
          maxLength: numberPayload.value ?? numberPayload.default,
        },
      };
    // Following rules require NUMBER component.
    case "statement.insert.row-limit":
    case "statement.affected-row-limit":
    case "column.maximum-character-length":
    case "column.maximum-varchar-length":
    case "column.auto-increment-initial-value":
    case "index.key-number-limit":
    case "index.total-number-limit":
    case "system.comment.length":
    case "advice.online-migration":
    case "table.text-fields-total-length":
    case "statement.where.maximum-logical-operator-count":
    case "statement.maximum-limit-value":
    case "statement.maximum-join-table-count":
    case "statement.maximum-statements-in-transaction":
      if (!numberPayload) {
        throw new Error(`Invalid rule ${template.type}`);
      }
      return {
        ...base,
        payload: {
          number: numberPayload.value ?? numberPayload.default,
        },
      };
  }

  throw new Error(`Invalid rule ${template.type}`);
};

// The convertRuleTemplateToPolicyRule will convert rule template to review policy rule for backend useage.
// Will throw exception if we don't implement the payload handler for specific type of rule.
export const convertRuleTemplateToPolicyRule = (
  rule: RuleTemplate
): SchemaPolicyRule[] => {
  const baseList: SchemaPolicyRule[] = rule.engineList.map((engine) => ({
    type: rule.type,
    level: rule.level,
    engine,
    comment: rule.comment ?? "",
  }));
  if ((rule.componentList?.length ?? 0) === 0) {
    return baseList;
  }

  return baseList.map((base) => {
    const result = mergeIndividualConfigAsRule(base, rule);
    const individualConfig = (rule.individualConfigList || []).find(
      (config) => config.engine === base.engine
    );
    if (individualConfig && result.payload) {
      for (const [key, val] of Object.entries(individualConfig.payload)) {
        (result.payload as any)[key] = val.value ?? val.default;
      }
    }
    return result;
  });
};

export const getRuleLocalizationKey = (type: string): string => {
  return type.split(".").join("-");
};

export const getRuleLocalization = (
  type: string
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
  ruleType: string,
  subscriptionPlan: PlanType
): boolean => {
  return true;
};
