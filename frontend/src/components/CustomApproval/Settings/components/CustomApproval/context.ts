import { type InjectionKey, inject, provide, type Ref } from "vue";

export type CustomApprovalContext = {
  hasFeature: Ref<boolean>;
  showFeatureModal: Ref<boolean>;
  allowAdmin: Ref<boolean>;
  ready: Ref<boolean>;
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
