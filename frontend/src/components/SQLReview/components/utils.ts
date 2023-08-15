import { groupBy, cloneDeep } from "lodash-es";
import {
  convertPolicyRuleToRuleTemplate,
  RuleConfigComponent,
  RuleTemplate,
  RuleType,
  SQLReviewPolicy,
  TEMPLATE_LIST,
  IndividualConfigForEngine,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import { PayloadForEngine } from "./RuleConfigComponents";

export const templateIdForEnvironment = (environment: Environment): string => {
  return `bb.sql-review.environment-policy.${environment.name}`;
};

export const rulesToTemplate = (
  review: SQLReviewPolicy,
  withDisabled = false
) => {
  const ruleTemplateList: RuleTemplate[] = [];
  const ruleTemplateMap: Map<RuleType, RuleTemplate> = TEMPLATE_LIST.reduce(
    (map, template) => {
      for (const rule of template.ruleList) {
        map.set(rule.type, rule);
      }
      return map;
    },
    new Map<RuleType, RuleTemplate>()
  );

  const groupByRule = groupBy(review.ruleList, (rule) => rule.type);

  for (const [type, ruleList] of Object.entries(groupByRule)) {
    const rule = ruleTemplateMap.get(type as RuleType);
    if (!rule) {
      continue;
    }
    if (rule.level === SQLReviewRuleLevel.DISABLED && !withDisabled) {
      continue;
    }
    const data = convertPolicyRuleToRuleTemplate(ruleList, rule);
    if (data) {
      ruleTemplateList.push(data);
    }
  }

  if (withDisabled) {
    for (const rule of ruleTemplateMap.values()) {
      ruleTemplateList.push({
        ...rule,
        level: SQLReviewRuleLevel.DISABLED,
      });
    }
  }

  return {
    id: templateIdForEnvironment(review.environment),
    review,
    ruleList: ruleTemplateList,
  };
};

export const payloadValueListToComponentList = (
  rule: RuleTemplate,
  data: PayloadForEngine
) => {
  const allEnginePayload = data.get(Engine.ENGINE_UNSPECIFIED) || [];
  const componentList = rule.componentList.reduce<RuleConfigComponent[]>(
    (list, component, index) => {
      switch (component.payload.type) {
        case "STRING_ARRAY":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: allEnginePayload[index] as string[],
            },
          });
          break;
        case "NUMBER":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: allEnginePayload[index] as number,
            },
          });
          break;
        case "BOOLEAN":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: allEnginePayload[index] as boolean,
            },
          });
          break;
        default:
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: allEnginePayload[index] as string,
            },
          });
          break;
      }
      return list;
    },
    []
  );

  const individualConfigList: IndividualConfigForEngine[] = [];
  for (const individualConfig of rule.individualConfigList) {
    const payloadList = data.get(individualConfig.engine) ?? [];
    const payload = cloneDeep(individualConfig.payload);

    for (const [index, component] of rule.componentList.entries()) {
      payload[component.key] = {
        default: payload[component.key].default,
        value: payloadList[index],
      };
    }

    individualConfigList.push({
      engine: individualConfig.engine,
      payload,
    });
  }

  return { componentList, individualConfigList };
};
