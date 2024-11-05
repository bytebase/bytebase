import type { InjectionKey, Ref } from "vue";
import { inject, provide } from "vue";

export type EditorContext = {
  statement: Ref<string>;
  setStatement: (statement: string) => void;
};

export const KEY = Symbol(
  "bb.issue.context.editor-view"
) as InjectionKey<EditorContext>;

export const useEditorContext = () => {
  return inject(KEY)!;
};

export const provideEditorContext = (context: EditorContext) => {
  provide(KEY, context);
  return context;
};
