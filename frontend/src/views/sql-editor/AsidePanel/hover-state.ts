import type { Ref } from "vue";
import { inject, provide, ref } from "vue";
import { useDelayedValue } from "@/composables/useDelayedValue";
import type { Position } from "@/types";

type UpdateFn<T> = ReturnType<typeof useDelayedValue<T | undefined>>["update"];

export type HoverStateContext<S> = {
  state: Ref<S | undefined>;
  position: Ref<Position>;
  update: UpdateFn<S>;
};

export const useHoverStateContext = <S>(key: string) => {
  const KEY = `bb.sql-editor.${key}.hover-state`;
  return inject(KEY)! as HoverStateContext<S>;
};

export const provideHoverStateContext = <S>(key: string) => {
  const { value: state, update } = useDelayedValue<S | undefined>(undefined, {
    delayBefore: 500,
    delayAfter: 500,
  });
  const position = ref<Position>({
    x: 0,
    y: 0,
  });
  const context: HoverStateContext<S> = {
    state,
    position,
    update,
  };

  const KEY = `bb.sql-editor.${key}.hover-state`;
  provide(KEY, context);

  return context;
};
