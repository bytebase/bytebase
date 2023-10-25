import { InjectionKey, Ref, inject, provide } from "vue";
import { useDelayedValue } from "@/composables/useDelayedValue";
import { SQLEditorTreeNode } from "@/types";

type UpdateFn = ReturnType<
  typeof useDelayedValue<SQLEditorTreeNode | undefined>
>["update"];

export type HoverStateContext = {
  node: Ref<SQLEditorTreeNode | undefined>;
  update: UpdateFn;
};

export const KEY = Symbol(
  "bb.sql-editor.tree.hover-state"
) as InjectionKey<HoverStateContext>;

export const useHoverStateContext = () => {
  return inject(KEY)!;
};

export const provideHoverStateContext = () => {
  const { value: node, update } = useDelayedValue<
    SQLEditorTreeNode | undefined
  >(undefined, {
    delayBefore: 500,
    delayAfter: 500,
  });
  const context: HoverStateContext = {
    node,
    update,
  };

  provide(KEY, context);

  return context;
};
