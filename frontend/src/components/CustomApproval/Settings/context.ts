import { inject, provide, type InjectionKey, type Ref } from "vue";
import { Risk } from "@/types/proto/v1/risk_service";
import { ApprovalConfigSetting_Rule as ApprovalRule } from "@/types/proto/store/setting";

export const TabViewList = ["risk", "workflow"] as const;

export type TabView = typeof TabViewList[number];

export type RiskDialogContext = {
  mode: "EDIT" | "CREATE";
  risk: Risk;
};

export type ApprovalDialogContext = {
  mode: "EDIT" | "CREATE";
  rule: ApprovalRule;
};

export type ApprovalConfigContext = {
  searchText: string;
};

export type CustomApprovalSettingsContext = {
  hasFeature: Ref<boolean>;
  showFeatureModal: Ref<boolean>;
  allowAdmin: Ref<boolean>;
  tabView: Ref<TabView>;
  ready: Ref<boolean>;

  approvalConfigContext: Ref<ApprovalConfigContext>;
  riskDialogContext: Ref<RiskDialogContext | undefined>;
  approvalDialogContext: Ref<ApprovalDialogContext | undefined>;
};

export const KEY = Symbol(
  "bb.custom-approval.settings"
) as InjectionKey<CustomApprovalSettingsContext>;

export const useCustomApprovalSettingsContext = () => {
  return inject(KEY)!;
};

export const provideCustomApprovalSettingsContext = (
  context: CustomApprovalSettingsContext
) => {
  provide(KEY, context);
};
