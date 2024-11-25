import Emittery from "emittery";
import { inject, provide } from "vue";

export const KEY = Symbol("bb.sql-editor.settings");

// eslint-disable-next-line @typescript-eslint/no-empty-object-type
export type SettingsEvents = Emittery<{}>;

// eslint-disable-next-line @typescript-eslint/no-empty-object-type
export const provideSettingsContext = (params: {}) => {
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
