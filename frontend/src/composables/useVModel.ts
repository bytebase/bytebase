import { watch } from "vue";

export const useVModel = <
  PS extends { [key in K]: PS[K] },
  K extends string,
  E extends (name: `update:${K}`, value: PS[K]) => void
>(
  props: PS,
  state: PS,
  emit: E,
  key: K,
  immediate = false
) => {
  watch(
    () => props[key],
    (v) => {
      state[key] = v;
    },
    { immediate }
  );

  watch(
    () => state[key],
    (v) => emit(`update:${key}`, v)
  );
};
