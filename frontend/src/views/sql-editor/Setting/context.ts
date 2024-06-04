import Emittery from "emittery";
import { inject, provide } from "vue";

export const KEY = Symbol("bb.sql-editor.settings");

export type SettingsEvents = Emittery<{
  // empty by now
}>;

export const provideSettingsContext = (params: {
  // empty by now
}) => {
  const events = new Emittery() as SettingsEvents;

  const context = {
    events,
    ...params,
  };

  provide(KEY, context);

  return context;
};

export type SettingsContext = ReturnType<typeof provideSettingsContext>;

export const useSettingsContext = () => {
  return inject(KEY) as SettingsContext;
};
