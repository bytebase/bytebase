import { inject, provide, type InjectionKey, type Ref } from "vue";
import { Risk } from "@/types/proto/v1/risk_service";

export type DialogContext = {
  mode: "EDIT" | "CREATE";
  risk: Risk;
};

export type RiskCenterContext = {
  hasFeature: Ref<boolean>;
  showFeatureModal: Ref<boolean>;
  allowAdmin: Ref<boolean>;
  ready: Ref<boolean>;

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
