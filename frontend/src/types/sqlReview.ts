import { create } from "@bufbuild/protobuf";
import { t, te } from "@/plugins/i18n";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { SQLReviewRule } from "@/types/proto-es/v1/review_config_service_pb";
import {
  SQLReviewRule_CommentConventionRulePayloadSchema,
  SQLReviewRule_Level,
  SQLReviewRule_NamingCaseRulePayloadSchema,
  SQLReviewRule_NamingRulePayloadSchema,
  SQLReviewRule_NumberRulePayloadSchema,
  SQLReviewRule_StringArrayRulePayloadSchema,
  SQLReviewRule_StringRulePayloadSchema,
  SQLReviewRule_Type,
  SQLReviewRuleSchema,
} from "@/types/proto-es/v1/review_config_service_pb";
import sqlReviewDevTemplate from "./sql-review.dev.yaml";
import sqlReviewProdTemplate from "./sql-review.prod.yaml";
import sqlReviewSampleTemplate from "./sql-review.sample.yaml";
import sqlReviewSchema from "./sql-review-schema.yaml";

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

// SchemaPolicyRule is an alias for the proto SQLReviewRule type.
// We use the proto type directly instead of maintaining a separate interface.
export type SchemaPolicyRule = SQLReviewRule;

// The API for SQL review policy in backend.
export interface SQLReviewPolicy {
  id: string;

  // Standard fields
  // enforce means if the policy is active
  enforce: boolean;

  // Domain specific fields
  name: string;
  ruleList: SchemaPolicyRule[];
  resources: string[];
}

// type, engine, and level are enum strings.
// we need to convert them first.
interface RuleTemplateV2Raw {
  type: string; // keyof typeof SQLReviewRule_Type
  category: string;
  engine: string; // keyof typeof Engine
  level: string; // keyof typeof SQLReviewRule_Level
  componentList: RuleConfigComponent[];
}

// RuleTemplateV2 is the rule template. Used by the frontend
export interface RuleTemplateV2 {
  type: SQLReviewRule_Type;
  category: string;
  engine: Engine;
  level: SQLReviewRule_Level;
  componentList: RuleConfigComponent[];
}

// SQLReviewPolicyTemplateV2 is the rule template set
export interface SQLReviewPolicyTemplateV2 {
  id: string;
  ruleList: RuleTemplateV2[];
}

// Helper to convert SQLReviewRule_Type enum value to string key
export const ruleTypeToString = (type: SQLReviewRule_Type): string => {
  return SQLReviewRule_Type[type] as string;
};

export const getRuleMapByEngine = (
  ruleList: RuleTemplateV2[]
): Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>> => {
  return ruleList.reduce((map, rule) => {
    const engine = rule.engine; // Engine is already numeric enum, no conversion needed
    if (!map.has(engine)) {
      map.set(engine, new Map());
    }
    map.get(engine)?.set(rule.type, {
      ...rule,
      level: rule.level,
      engine,
      componentList: rule.componentList || [],
    });
    return map;
  }, new Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>());
};

const convertRuleTemplateV2Raw = (
  sqlReviewSchema: RuleTemplateV2Raw[]
): RuleTemplateV2[] => {
  return sqlReviewSchema.map((rawRule) => {
    // Convert type string key to enum value
    const typeKey = rawRule.type as keyof typeof SQLReviewRule_Type;
    const type =
      SQLReviewRule_Type[typeKey] ?? SQLReviewRule_Type.TYPE_UNSPECIFIED;

    // Convert engine string key to enum value
    const engineKey = rawRule.engine as keyof typeof Engine;
    const engine = Engine[engineKey] ?? Engine.ENGINE_UNSPECIFIED;

    // Schema rules from YAML don't have levels (enforced by tests).
    // WARNING is the default severity when users add these rules to their policies.
    const level = SQLReviewRule_Level.WARNING;

    return {
      ...rawRule,
      type,
      engine,
      level,
    };
  });
};

export const ruleTemplateMapV2 = getRuleMapByEngine(
  convertRuleTemplateV2Raw(sqlReviewSchema as unknown as RuleTemplateV2Raw[])
);

// Build the frontend template list based on schema and template.
export const TEMPLATE_LIST_V2: SQLReviewPolicyTemplateV2[] = (function () {
  interface PayloadObject {
    [key: string]: unknown;
  }
  const templateList = [
    sqlReviewSampleTemplate,
    sqlReviewDevTemplate,
    sqlReviewProdTemplate,
  ] as {
    id: string;
    ruleList: {
      type: string;
      level: string; // keyof typeof SQLReviewRule_Level
      payload?: PayloadObject;
      engine: string; // keyof typeof Engine
    }[];
  }[];

  const resp = templateList.map((template) => {
    const ruleList: RuleTemplateV2[] = [];

    for (const rule of template.ruleList) {
      // Convert type string to enum for map lookup
      const typeKey = rule.type as keyof typeof SQLReviewRule_Type;
      const type =
        SQLReviewRule_Type[typeKey] ?? SQLReviewRule_Type.TYPE_UNSPECIFIED;

      // Convert engine string to enum for map lookup
      const engineKey = rule.engine as keyof typeof Engine;
      const engine = Engine[engineKey] ?? Engine.ENGINE_UNSPECIFIED;

      const ruleTemplate = ruleTemplateMapV2.get(engine)?.get(type);
      if (!ruleTemplate) {
        continue;
      }

      // Convert level string to enum - all template rules must have a level
      if (!rule.level) {
        throw new Error(
          `Template rule ${rule.type} is missing required 'level' field in template ${template.id}`
        );
      }
      const levelKey = rule.level as keyof typeof SQLReviewRule_Level;
      const level = SQLReviewRule_Level[levelKey];
      if (level === undefined) {
        throw new Error(
          `Template rule ${rule.type} has invalid level '${rule.level}' in template ${template.id}`
        );
      }

      ruleList.push({
        ...ruleTemplate,
        level,
        // Using template rule payload to override the component list.
        componentList: ruleTemplate.componentList.map((component) => {
          if (rule.payload && rule.payload[component.key] !== undefined) {
            const payloadValue = rule.payload[component.key];
            return {
              ...component,
              payload: {
                ...component.payload,
                default: payloadValue,
              },
            } as RuleConfigComponent;
          }
          return component;
        }),
      });
    }

    return {
      id: template.id,
      ruleList,
    };
  });

  resp.unshift({
    id: "bb.sql-review.empty",
    ruleList: [],
  });

  return resp;
})();

// convertToCategoryMap will reduce RuleTemplate list to map by category.
export const convertToCategoryMap = (
  ruleList: RuleTemplateV2[]
): Map<string, RuleTemplateV2[]> => {
  return ruleList.reduce((map, rule) => {
    if (!map.has(rule.category)) {
      map.set(rule.category, []);
    }
    map.get(rule.category)?.push(rule);
    return map;
  }, new Map<string, RuleTemplateV2[]>());
};

// The convertPolicyRuleToRuleTemplate will convert the review policy rule to rule template for frontend useage.
// Will throw exception if we don't implement the payload handler for specific type of rule.
export const convertPolicyRuleToRuleTemplate = (
  policyRule: SchemaPolicyRule,
  ruleTemplate: RuleTemplateV2
): RuleTemplateV2 => {
  if (policyRule.type !== ruleTemplate.type) {
    throw new Error(
      `The rule type is not same. policyRule:${policyRule.type}, ruleTemplate:${ruleTemplate.type}`
    );
  }

  const res: RuleTemplateV2 = {
    ...ruleTemplate,
    engine: policyRule.engine,
    level: policyRule.level,
  };

  const componentList = ruleTemplate.componentList ?? [];
  if (componentList.length === 0) {
    return res;
  }

  // Extract payload value from proto oneof
  let payload: Record<string, unknown> = {};
  if (policyRule?.payload?.case) {
    payload = policyRule.payload.value as Record<string, unknown>;
  }

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
  const stringArrayComponent = componentList.find(
    (c) => c.payload.type === "STRING_ARRAY"
  );

  switch (ruleTemplate.type) {
    case SQLReviewRule_Type.STATEMENT_QUERY_MINIMUM_PLAN_LEVEL:
      if (!stringComponent) {
        throw new Error(`Invalid rule ${ruleTypeToString(ruleTemplate.type)}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...stringComponent,
            payload: {
              ...stringComponent.payload,
              value: payload.value, // proto field is 'value'
            } as StringPayload,
          },
        ],
      };
    // Following rules require STRING component.
    case SQLReviewRule_Type.TABLE_DROP_NAMING_CONVENTION:
      if (!stringComponent) {
        throw new Error(`Invalid rule ${ruleTypeToString(ruleTemplate.type)}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...stringComponent,
            payload: {
              ...stringComponent.payload,
              value: payload.format,
            } as StringPayload,
          },
        ],
      };
    // Following rules require STRING and NUMBER component.
    case SQLReviewRule_Type.NAMING_COLUMN:
    case SQLReviewRule_Type.NAMING_COLUMN_AUTO_INCREMENT:
    case SQLReviewRule_Type.NAMING_TABLE:
      if (!stringComponent || !numberComponent) {
        throw new Error(`Invalid rule ${ruleTypeToString(ruleTemplate.type)}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...stringComponent,
            payload: {
              ...stringComponent.payload,
              value: payload.format,
            } as StringPayload,
          },
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: payload.maxLength,
            } as NumberPayload,
          },
        ],
      };
    // Following rules require TEMPLATE component.
    case SQLReviewRule_Type.NAMING_INDEX_PK:
      if (!templateComponent) {
        throw new Error(`Invalid rule ${ruleTypeToString(ruleTemplate.type)}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...templateComponent,
            payload: {
              ...templateComponent.payload,
              value: payload.format,
            } as TemplatePayload,
          },
        ],
      };
    // Following rules require TEMPLATE and NUMBER component.
    case SQLReviewRule_Type.NAMING_INDEX_IDX:
    case SQLReviewRule_Type.NAMING_INDEX_UK:
    case SQLReviewRule_Type.NAMING_INDEX_FK:
      if (!templateComponent || !numberComponent) {
        throw new Error(`Invalid rule ${ruleTypeToString(ruleTemplate.type)}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...templateComponent,
            payload: {
              ...templateComponent.payload,
              value: payload.format,
            } as TemplatePayload,
          },
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: payload.maxLength,
            } as NumberPayload,
          },
        ],
      };
    // Following rules require BOOLEAN component.
    case SQLReviewRule_Type.NAMING_IDENTIFIER_CASE:
      if (!booleanComponent) {
        throw new Error(`Invalid rule ${ruleTypeToString(ruleTemplate.type)}`);
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
    case SQLReviewRule_Type.COLUMN_REQUIRED: {
      const requiredColumnComponent = componentList[0];
      // Proto uses stringArrayPayload with 'list' field
      const value = payload.list || [];
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
    // Following rules require STRING_ARRAY component.
    case SQLReviewRule_Type.COLUMN_TYPE_DISALLOW_LIST:
    case SQLReviewRule_Type.INDEX_PRIMARY_KEY_TYPE_ALLOWLIST:
    case SQLReviewRule_Type.INDEX_TYPE_ALLOW_LIST:
    case SQLReviewRule_Type.SYSTEM_CHARSET_ALLOWLIST:
    case SQLReviewRule_Type.SYSTEM_COLLATION_ALLOWLIST:
    case SQLReviewRule_Type.SYSTEM_FUNCTION_DISALLOWED_LIST:
    case SQLReviewRule_Type.TABLE_DISALLOW_DML:
    case SQLReviewRule_Type.TABLE_DISALLOW_DDL: {
      if (!stringArrayComponent) {
        throw new Error(`Invalid rule ${ruleTypeToString(ruleTemplate.type)}`);
      }
      const stringArrayPayload = {
        ...stringArrayComponent.payload,
        value: payload.list || [],
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
    // Following rules require BOOLEAN and NUMBER component.
    case SQLReviewRule_Type.COLUMN_COMMENT:
    case SQLReviewRule_Type.TABLE_COMMENT: {
      const requireComponent = componentList.find((c) => c.key === "required");

      if (!requireComponent || !numberComponent) {
        throw new Error(`Invalid rule ${ruleTypeToString(ruleTemplate.type)}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...requireComponent,
            payload: {
              ...requireComponent.payload,
              value: payload.required,
            } as BooleanPayload,
          },
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: payload.maxLength,
            } as NumberPayload,
          },
        ],
      };
    }
    // Following rules require NUMBER component.
    case SQLReviewRule_Type.STATEMENT_INSERT_ROW_LIMIT:
    case SQLReviewRule_Type.STATEMENT_AFFECTED_ROW_LIMIT:
    case SQLReviewRule_Type.COLUMN_MAXIMUM_CHARACTER_LENGTH:
    case SQLReviewRule_Type.COLUMN_MAXIMUM_VARCHAR_LENGTH:
    case SQLReviewRule_Type.COLUMN_AUTO_INCREMENT_INITIAL_VALUE:
    case SQLReviewRule_Type.INDEX_KEY_NUMBER_LIMIT:
    case SQLReviewRule_Type.INDEX_TOTAL_NUMBER_LIMIT:
    case SQLReviewRule_Type.SYSTEM_COMMENT_LENGTH:
    case SQLReviewRule_Type.ADVICE_ONLINE_MIGRATION:
    case SQLReviewRule_Type.TABLE_TEXT_FIELDS_TOTAL_LENGTH:
    case SQLReviewRule_Type.TABLE_LIMIT_SIZE:
    case SQLReviewRule_Type.STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT:
    case SQLReviewRule_Type.STATEMENT_MAXIMUM_LIMIT_VALUE:
    case SQLReviewRule_Type.STATEMENT_MAXIMUM_JOIN_TABLE_COUNT:
    case SQLReviewRule_Type.STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION:
      if (!numberComponent) {
        throw new Error(`Invalid rule ${ruleTypeToString(ruleTemplate.type)}`);
      }

      return {
        ...res,
        componentList: [
          {
            ...numberComponent,
            payload: {
              ...numberComponent.payload,
              value: payload.number,
            } as NumberPayload,
          },
        ],
      };
  }

  throw new Error(`Invalid rule ${ruleTemplate.type}`);
};

const mergeIndividualConfigAsRule = (
  base: SchemaPolicyRule,
  template: RuleTemplateV2
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
    case SQLReviewRule_Type.STATEMENT_QUERY_MINIMUM_PLAN_LEVEL:
      if (!stringPayload) {
        throw new Error(`Invalid rule ${ruleTypeToString(template.type)}`);
      }

      return {
        ...base,
        payload: {
          case: "stringPayload",
          value: create(SQLReviewRule_StringRulePayloadSchema, {
            value: stringPayload.value ?? stringPayload.default,
          }),
        },
      };
    // Following rules require STRING component.
    case SQLReviewRule_Type.TABLE_DROP_NAMING_CONVENTION:
      if (!stringPayload) {
        throw new Error(`Invalid rule ${ruleTypeToString(template.type)}`);
      }

      return {
        ...base,
        payload: {
          case: "namingPayload",
          value: create(SQLReviewRule_NamingRulePayloadSchema, {
            format: stringPayload.value ?? stringPayload.default,
            maxLength: 0,
          }),
        },
      };
    // Following rules require STRING and NUMBER component.
    case SQLReviewRule_Type.NAMING_COLUMN:
    case SQLReviewRule_Type.NAMING_COLUMN_AUTO_INCREMENT:
    case SQLReviewRule_Type.NAMING_TABLE:
      if (!stringPayload || !numberPayload) {
        throw new Error(`Invalid rule ${ruleTypeToString(template.type)}`);
      }

      return {
        ...base,
        payload: {
          case: "namingPayload",
          value: create(SQLReviewRule_NamingRulePayloadSchema, {
            format: stringPayload.value ?? stringPayload.default,
            maxLength: numberPayload.value ?? numberPayload.default,
          }),
        },
      };
    // Following rules require TEMPLATE component.
    case SQLReviewRule_Type.NAMING_INDEX_PK:
      if (!templatePayload) {
        throw new Error(`Invalid rule ${ruleTypeToString(template.type)}`);
      }
      return {
        ...base,
        payload: {
          case: "namingPayload",
          value: create(SQLReviewRule_NamingRulePayloadSchema, {
            format: templatePayload.value ?? templatePayload.default,
            maxLength: 0,
          }),
        },
      };
    // Following rules require TEMPLATE and NUMBER component.
    case SQLReviewRule_Type.NAMING_INDEX_IDX:
    case SQLReviewRule_Type.NAMING_INDEX_UK:
    case SQLReviewRule_Type.NAMING_INDEX_FK:
      if (!templatePayload || !numberPayload) {
        throw new Error(`Invalid rule ${ruleTypeToString(template.type)}`);
      }
      return {
        ...base,
        payload: {
          case: "namingPayload",
          value: create(SQLReviewRule_NamingRulePayloadSchema, {
            format: templatePayload.value ?? templatePayload.default,
            maxLength: numberPayload.value ?? numberPayload.default,
          }),
        },
      };
    // Following rules require BOOLEAN component.
    case SQLReviewRule_Type.NAMING_IDENTIFIER_CASE:
      if (!booleanPayload) {
        throw new Error(`Invalid rule ${ruleTypeToString(template.type)}`);
      }
      return {
        ...base,
        payload: {
          case: "namingCasePayload",
          value: create(SQLReviewRule_NamingCaseRulePayloadSchema, {
            upper: booleanPayload.value ?? booleanPayload.default,
          }),
        },
      };
    // Following rules require STRING_ARRAY component.
    case SQLReviewRule_Type.COLUMN_REQUIRED:
    case SQLReviewRule_Type.COLUMN_TYPE_DISALLOW_LIST:
    case SQLReviewRule_Type.INDEX_PRIMARY_KEY_TYPE_ALLOWLIST:
    case SQLReviewRule_Type.INDEX_TYPE_ALLOW_LIST:
    case SQLReviewRule_Type.SYSTEM_CHARSET_ALLOWLIST:
    case SQLReviewRule_Type.SYSTEM_COLLATION_ALLOWLIST:
    case SQLReviewRule_Type.SYSTEM_FUNCTION_DISALLOWED_LIST:
    case SQLReviewRule_Type.TABLE_DISALLOW_DML:
    case SQLReviewRule_Type.TABLE_DISALLOW_DDL: {
      if (!stringArrayPayload) {
        throw new Error(`Invalid rule ${ruleTypeToString(template.type)}`);
      }
      return {
        ...base,
        payload: {
          case: "stringArrayPayload",
          value: create(SQLReviewRule_StringArrayRulePayloadSchema, {
            list: stringArrayPayload.value ?? stringArrayPayload.default,
          }),
        },
      };
    }
    // Following rules require BOOLEAN and NUMBER component.
    case SQLReviewRule_Type.COLUMN_COMMENT:
    case SQLReviewRule_Type.TABLE_COMMENT: {
      const requirePayload = componentList.find((c) => c.key === "required")
        ?.payload as BooleanPayload | undefined;

      if (!requirePayload || !numberPayload) {
        throw new Error(`Invalid rule ${ruleTypeToString(template.type)}`);
      }
      return {
        ...base,
        payload: {
          case: "commentConventionPayload",
          value: create(SQLReviewRule_CommentConventionRulePayloadSchema, {
            required: requirePayload.value ?? requirePayload.default,
            maxLength: numberPayload.value ?? numberPayload.default,
          }),
        },
      };
    }
    // Following rules require NUMBER component.
    case SQLReviewRule_Type.STATEMENT_INSERT_ROW_LIMIT:
    case SQLReviewRule_Type.STATEMENT_AFFECTED_ROW_LIMIT:
    case SQLReviewRule_Type.COLUMN_MAXIMUM_CHARACTER_LENGTH:
    case SQLReviewRule_Type.COLUMN_MAXIMUM_VARCHAR_LENGTH:
    case SQLReviewRule_Type.COLUMN_AUTO_INCREMENT_INITIAL_VALUE:
    case SQLReviewRule_Type.INDEX_KEY_NUMBER_LIMIT:
    case SQLReviewRule_Type.INDEX_TOTAL_NUMBER_LIMIT:
    case SQLReviewRule_Type.SYSTEM_COMMENT_LENGTH:
    case SQLReviewRule_Type.ADVICE_ONLINE_MIGRATION:
    case SQLReviewRule_Type.TABLE_TEXT_FIELDS_TOTAL_LENGTH:
    case SQLReviewRule_Type.TABLE_LIMIT_SIZE:
    case SQLReviewRule_Type.STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT:
    case SQLReviewRule_Type.STATEMENT_MAXIMUM_LIMIT_VALUE:
    case SQLReviewRule_Type.STATEMENT_MAXIMUM_JOIN_TABLE_COUNT:
    case SQLReviewRule_Type.STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION:
      if (!numberPayload) {
        throw new Error(`Invalid rule ${ruleTypeToString(template.type)}`);
      }
      return {
        ...base,
        payload: {
          case: "numberPayload",
          value: create(SQLReviewRule_NumberRulePayloadSchema, {
            number: numberPayload.value ?? numberPayload.default,
          }),
        },
      };
  }

  throw new Error(`Invalid rule ${template.type}`);
};

export const convertRuleMapToPolicyRuleList = (
  ruleMapByEngine: Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>
): SchemaPolicyRule[] => {
  const resp: SchemaPolicyRule[] = [];

  for (const mapByType of ruleMapByEngine.values()) {
    for (const rule of mapByType.values()) {
      resp.push(convertRuleTemplateToPolicyRule(rule));
    }
  }

  return resp;
};

// The convertRuleTemplateToPolicyRule will convert rule template to review policy rule for backend useage.
// Will throw exception if we don't implement the payload handler for specific type of rule.
const convertRuleTemplateToPolicyRule = (
  rule: RuleTemplateV2
): SchemaPolicyRule => {
  const base: SchemaPolicyRule = create(SQLReviewRuleSchema, {
    type: rule.type,
    level: rule.level,
    engine: rule.engine,
    // payload will be set by mergeIndividualConfigAsRule if componentList is not empty
  });
  if ((rule.componentList?.length ?? 0) === 0) {
    return base;
  }

  return mergeIndividualConfigAsRule(base, rule);
};

export const getRuleLocalizationKey = (type: string): string => {
  // Return the SCREAMING_SNAKE_CASE format as-is
  // This matches the keys in the locale files (e.g., "TABLE_REQUIRE_PK")
  return type;
};

export const getRuleLocalization = (
  type: string,
  engine?: Engine
): { title: string; description: string } => {
  const key = getRuleLocalizationKey(type);
  let title = t(`sql-review.rule.${key}.title`);
  let description = t(`sql-review.rule.${key}.description`);

  if (engine) {
    const engineSpecificKey = `${key}.${Engine[engine].toLowerCase()}`;
    if (te(`sql-review.rule.${engineSpecificKey}.title`)) {
      title = t(`sql-review.rule.${engineSpecificKey}.title`);
    }
    if (te(`sql-review.rule.${engineSpecificKey}.description`)) {
      description = t(`sql-review.rule.${engineSpecificKey}.description`);
    }
  }

  return {
    title,
    description,
  };
};
