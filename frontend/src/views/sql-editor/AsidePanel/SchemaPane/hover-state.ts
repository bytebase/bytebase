import type { InjectionKey, Ref } from "vue";
import { inject, provide } from "vue";
import { useDelayedValue } from "@/composables/useDelayedValue";
import type { TreeNode } from "./common";

type UpdateFn = ReturnType<
  typeof useDelayedValue<TreeNode | undefined>
>["update"];

export type HoverStateContext = {
  node: Ref<TreeNode | undefined>;
  update: UpdateFn;
};

export const KEY = Symbol(
  "bb.sql-editor.tree.hover-state"
) as InjectionKey<HoverStateContext>;

export const useHoverStateContext = () => {
  return inject(KEY)!;
};

export const provideHoverStateContext = () => {
  const { value: node, update } = useDelayedValue<TreeNode | undefined>(
    undefined,
    {
      delayBefore: 500,
      delayAfter: 500,
    }
  );
  const context: HoverStateContext = {
    node,
    update,
  };

  provide(KEY, context);

  return context;
};
