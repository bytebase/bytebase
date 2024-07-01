import type {
  RuleConfigComponent,
  RuleTemplateV2,
  SQLReviewPolicy,
} from "@/types";
import { convertPolicyRuleToRuleTemplate, ruleTemplateMapV2 } from "@/types";
import type { PayloadValueType } from "./RuleConfigComponents";

export const rulesToTemplate = (review: SQLReviewPolicy) => {
  const ruleTemplateList: RuleTemplateV2[] = [];

  for (const rule of review.ruleList) {
    const ruleTemplate = ruleTemplateMapV2.get(rule.type)?.get(rule.engine);
    if (!ruleTemplate) {
      continue;
    }

    const data = convertPolicyRuleToRuleTemplate(rule, ruleTemplate);
    if (data) {
      ruleTemplateList.push(data);
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
