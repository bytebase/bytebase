import { inject, provide, type InjectionKey, type Ref } from "vue";
import type { LocalApprovalRule } from "@/types";

export const TabValueList = ["rules", "flows"] as const;
export type TabValue = typeof TabValueList[number];

export type DialogContext = {
  mode: "EDIT" | "CREATE";
  rule: LocalApprovalRule;
};

export type CustomApprovalContext = {
  hasFeature: Ref<boolean>;
  showFeatureModal: Ref<boolean>;
  allowAdmin: Ref<boolean>;
  ready: Ref<boolean>;
  tab: Ref<TabValue>;

  dialog: Ref<DialogContext | undefined>;
};

export const KEY = Symbol(
  "bb.settings.custom-approval"
) as InjectionKey<CustomApprovalContext>;

export const useCustomApprovalContext = () => {
  return inject(KEY)!;
};

export const provideCustomApprovalContext = (
  context: CustomApprovalContext
) => {
  provide(KEY, context);
};
