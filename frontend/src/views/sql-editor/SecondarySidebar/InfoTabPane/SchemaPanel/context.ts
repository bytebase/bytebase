import { InjectionKey, inject, provide, Ref, ref } from "vue";

export type SchemaPanelContext = {
  keyword: Ref<string>;
};

export const KEY = Symbol(
  "bb.sql-editor.schema-panel"
) as InjectionKey<SchemaPanelContext>;

export const useSchemaPanelContext = () => {
  return inject(KEY)!;
};

export const provideSchemaPanelContext = () => {
  const context: SchemaPanelContext = {
    keyword: ref(""),
  };

  provide(KEY, context);

  return context;
};
