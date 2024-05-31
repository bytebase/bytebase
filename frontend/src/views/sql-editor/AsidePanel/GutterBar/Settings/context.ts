import Emittery from "emittery";
import { inject, provide, ref, type Ref } from "vue";
import type { Size } from "../common";

export const KEY = Symbol("bb.sql-editor.settings");

export type SettingsEvents = Emittery<{
  // empty by now
}>;

export const provideSettingsContext = (params: { size: Ref<Size> }) => {
  const events = new Emittery() as SettingsEvents;
  const showPanel = ref(false);

  const context = {
    events,
    ...params,
    showPanel,
  };

  provide(KEY, context);

  return context;
};

export type SettingsContext = ReturnType<typeof provideSettingsContext>;

export const useSettingsContext = () => {
  return inject(KEY) as SettingsContext;
};
