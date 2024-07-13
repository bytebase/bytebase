import type { RuleTemplateV2 } from "@/types/sqlReview";
import type { PayloadValueType } from "./types";

export const getRulePayload = (rule: RuleTemplateV2): PayloadValueType[] => {
  const { componentList } = rule;

  if (componentList.length === 0) {
    return [];
  }

  const basePayload = componentList.reduce<
    { key: string; value: PayloadValueType }[]
  >((list, component) => {
    list.push({
      key: component.key,
      value: component.payload.value ?? component.payload.default,
    });
    return list;
  }, []);

  return basePayload.map((val) => val.value);
};
