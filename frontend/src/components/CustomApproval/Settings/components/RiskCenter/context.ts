import { inject, provide, type InjectionKey, type Ref } from "vue";
import { Risk, Risk_Source } from "@/types/proto/v1/risk_service";

export type NavigationContext = {
  source: Risk_Source;
  levels: Set<number>; // empty to "ALL"
  search: string; // Risk_Source.SOURCE_UNSPECIFIED to "ALL"
};

export type DialogContext = {
  mode: "EDIT" | "CREATE";
  risk: Risk;
};

export type RiskCenterContext = {
  hasFeature: Ref<boolean>;
  showFeatureModal: Ref<boolean>;
  allowAdmin: Ref<boolean>;
  ready: Ref<boolean>;

  navigation: Ref<NavigationContext>;
  dialog: Ref<DialogContext | undefined>;
};

export const KEY = Symbol(
  "bb.settings.risk-center"
) as InjectionKey<RiskCenterContext>;

export const useRiskCenterContext = () => {
  return inject(KEY)!;
};

export const provideRiskCenterContext = (context: RiskCenterContext) => {
  provide(KEY, context);
};
