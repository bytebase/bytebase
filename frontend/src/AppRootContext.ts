import { InjectionKey, inject, provide, Ref, ref } from "vue";
import {
  useDatabaseV1Store,
  useInstanceV1Store,
  useProjectV1Store,
} from "./store";

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

export const restartAppRoot = () => {
  const { key } = appRootContextSingleton;
  key.value = key.value + 1;

  useDatabaseV1Store().reset();
  useProjectV1Store().reset();
  useInstanceV1Store().reset();
};
