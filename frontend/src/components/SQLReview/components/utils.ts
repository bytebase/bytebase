import type {
  RuleConfigComponent,
  RuleTemplateV2,
  SQLReviewPolicy,
} from "@/types";
import { convertPolicyRuleToRuleTemplate, ruleTemplateMapV2 } from "@/types";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import type { PayloadValueType } from "./RuleConfigComponents";

export const rulesToTemplate = (
  review: SQLReviewPolicy,
  withDisabled: boolean
) => {
  const ruleTemplateList: RuleTemplateV2[] = [];
  const usedRule = new Set();

  for (const rule of review.ruleList) {
    const ruleTemplate = ruleTemplateMapV2.get(rule.type)?.get(rule.engine);
    if (!ruleTemplate) {
      continue;
    }

    usedRule.add(rule.type);
    const data = convertPolicyRuleToRuleTemplate(rule, ruleTemplate);
    if (data) {
      ruleTemplateList.push(data);
    }
  }

  if (withDisabled) {
    for (const [key, map] of ruleTemplateMapV2.entries()) {
      if (usedRule.has(key)) {
        continue;
      }
      for (const rule of map.values()) {
        ruleTemplateList.push({
          ...rule,
          level: SQLReviewRuleLevel.DISABLED,
        });
      }
    }
  }

  return {
    id: `bb.sql-review.${review.id}`,
    review,
    ruleList: ruleTemplateList,
  };
};

export const payloadValueListToComponentList = (
  rule: RuleTemplateV2,
  data: PayloadValueType[]
) => {
  const componentList = rule.componentList.reduce<RuleConfigComponent[]>(
    (list, component, index) => {
      switch (component.payload.type) {
        case "STRING_ARRAY":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as string[],
            },
          });
          break;
        case "NUMBER":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as number,
            },
          });
          break;
        case "BOOLEAN":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as boolean,
            },
          });
          break;
        default:
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as string,
            },
          });
          break;
      }
      return list;
    },
    []
  );

  return { componentList };
};
