import { type InjectionKey, type Ref, inject, provide, ref } from "vue";
import { Risk_Source } from "@/types/proto/v1/risk_service";

export type RiskFilterContext = {
  source: Ref<Risk_Source>; // default Risk_Source.SOURCE_UNSPECIFIED to "ALL"
  levels: Ref<Set<number>>; // default empty to "ALL"
  search: Ref<string>; // default ""
};

export const KEY = Symbol(
  "bb.settings.custom-approval+risk-center.risk-filter"
) as InjectionKey<RiskFilterContext>;

const useRiskFilterContext = () => {
  return inject(KEY)!;
};

const provideRiskFilterContext = (context: RiskFilterContext) => {
  provide(KEY, context);
};

export const useRiskFilter = () => {
  const context = useRiskFilterContext();
  return context;
};

export const provideRiskFilter = () => {
  provideRiskFilterContext({
    source: ref(Risk_Source.SOURCE_UNSPECIFIED),
    levels: ref(new Set()),
    search: ref(""),
  });
};
