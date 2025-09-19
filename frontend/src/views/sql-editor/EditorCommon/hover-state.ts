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

export const useHoverStateContext = <S>(key: string): HoverStateContext<S> => {
  const KEY = `bb.sql-editor.${key}.hover-state`;
  return inject(KEY)!;
};

export const provideHoverStateContext = <S>(key: string) => {
  const { value: state, update } = useDelayedValue<S | undefined>(undefined, {
    delayBefore: 1000,
    delayAfter: 350,
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
