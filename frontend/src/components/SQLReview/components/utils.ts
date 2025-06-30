import type {
  RuleConfigComponent,
  RuleTemplateV2,
  SQLReviewPolicy,
} from "@/types";
import { convertPolicyRuleToRuleTemplate, ruleTemplateMapV2 } from "@/types";
import { SQLReviewRuleLevel } from "@/types/proto-es/v1/org_policy_service_pb";
import type { PayloadValueType } from "./RuleConfigComponents";

export const getRuleKey = (rule: RuleTemplateV2) =>
  `${rule.engine}:${rule.type}`;

export const getTemplateId = (review: SQLReviewPolicy) =>
  `bb.sql-review.${review.id}`;

export const rulesToTemplate = (
  review: SQLReviewPolicy,
  withDisabled: boolean
) => {
  const ruleTemplateList: RuleTemplateV2[] = [];
  const usedRule = new Set<string>();

  for (const rule of review.ruleList) {
    const ruleTemplate = ruleTemplateMapV2.get(rule.engine)?.get(rule.type);
    if (!ruleTemplate) {
      continue;
    }

    usedRule.add(`${rule.engine}-${rule.type}`);
    const data = convertPolicyRuleToRuleTemplate(rule, ruleTemplate);
    if (data) {
      ruleTemplateList.push(data);
    }
  }

  if (withDisabled) {
    for (const map of ruleTemplateMapV2.values()) {
      for (const rule of map.values()) {
        if (usedRule.has(`${rule.engine}-${rule.type}`)) {
          continue;
        }
        ruleTemplateList.push({
          ...rule,
          level: SQLReviewRuleLevel.DISABLED,
        });
      }
    }
  }

  return {
    id: getTemplateId(review),
    review,
    ruleList: ruleTemplateList,
  };
};

export const payloadValueListToComponentList = (
  rule: RuleTemplateV2,
  data: PayloadValueType[]
) => {
  return rule.componentList.reduce<RuleConfigComponent[]>(
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
};
