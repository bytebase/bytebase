import { inject, provide, type InjectionKey, type Ref } from "vue";
import type { LocalApprovalRule } from "@/types";
import { ExternalApprovalSetting_Node } from "@/types/proto/store/setting";

export const TabValueList = ["rules", "flows", "external-approval"] as const;
export type TabValue = typeof TabValueList[number];

export type DialogContext = {
  mode: "EDIT" | "CREATE";
  rule: LocalApprovalRule;
};

export type ExternalApprovalNodeContext = {
  mode: "EDIT" | "CREATE";
  node: ExternalApprovalSetting_Node;
};

export type CustomApprovalContext = {
  hasFeature: Ref<boolean>;
  showFeatureModal: Ref<boolean>;
  allowAdmin: Ref<boolean>;
  ready: Ref<boolean>;
  tab: Ref<TabValue>;

  dialog: Ref<DialogContext | undefined>;
  externalApprovalNodeContext: Ref<ExternalApprovalNodeContext | undefined>;
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
