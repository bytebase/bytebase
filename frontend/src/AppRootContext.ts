import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";

export type AppRootContext = {
  key: Ref<number>;
};

const appRootContextSingleton: AppRootContext = {
  key: ref(0),
};

export const KEY = Symbol("bb.app.root") as InjectionKey<AppRootContext>;

export const useAppRootContext = () => {
  return inject(KEY)!;
};

export const provideAppRootContext = () => {
  provide(KEY, appRootContextSingleton);

  return appRootContextSingleton;
};
