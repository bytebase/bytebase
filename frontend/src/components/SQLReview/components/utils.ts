import type {
  RuleConfigComponent,
  RuleTemplateV2,
  SQLReviewPolicy,
} from "@/types";
import { convertPolicyRuleToRuleTemplate, ruleTemplateMapV2 } from "@/types";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import type { PayloadValueType } from "./RuleConfigComponents";

export const getRuleKey = (rule: RuleTemplateV2) =>
  `${rule.engine}:${rule.type}`;

export const getTemplateId = (review: SQLReviewPolicy) =>
  `bb.sql-review.${review.id}`;

export const rulesToTemplate = (review: SQLReviewPolicy) => {
  const ruleTemplateList: RuleTemplateV2[] = [];

  for (const rule of review.ruleList) {
    // rule.type is already SQLReviewRule_Type enum
    const type = rule.type ?? SQLReviewRule_Type.TYPE_UNSPECIFIED;

    const ruleTemplate = ruleTemplateMapV2.get(rule.engine)?.get(type);
    if (!ruleTemplate) {
      continue;
    }

    const data = convertPolicyRuleToRuleTemplate(rule, ruleTemplate);
    if (data) {
      ruleTemplateList.push(data);
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
