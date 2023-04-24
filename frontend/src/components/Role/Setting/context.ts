import { type InjectionKey, type Ref, provide, inject } from "vue";

export type CustomRoleSettingContext = {
  hasCustomRoleFeature: Ref<boolean>;
  showFeatureModal: Ref<boolean>;
};

const KEY = Symbol(
  "bb.settings.custom-role"
) as InjectionKey<CustomRoleSettingContext>;

export const provideCustomRoleSettingContext = (
  context: CustomRoleSettingContext
) => {
  provide(KEY, context);
};

export const useCustomRoleSettingContext = () => {
  return inject(KEY)!;
};
