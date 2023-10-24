import { Ref, reactive, ref } from "vue";

export type UseDelayedValueOptions = {
  delayBefore?: number;
  delayAfter?: number;
};

export const useDelayedValue = <T>(
  initialValue: T,
  options: UseDelayedValueOptions = {}
) => {
  const { delayBefore = 0, delayAfter = 0 } = options;
  const valueRef = ref(initialValue) as Ref<T>;
  const state = reactive<{
    timer: ReturnType<typeof setTimeout> | undefined;
  }>({
    timer: undefined,
  });
  const cancel = () => {
    if (state.timer) {
      clearTimeout(state.timer);
      state.timer = undefined;
    }
  };
  const update = (
    value: T,
    direction: "before" | "after",
    overrideDelay: number | undefined = undefined
  ) => {
    const delay =
      overrideDelay ?? (direction === "before" ? delayBefore : delayAfter);
    cancel();
    if (delay) {
      state.timer = setTimeout(() => {
        valueRef.value = value;
      }, delay);
    } else {
      valueRef.value = value;
    }
  };

  return {
    value: valueRef,
    update,
  };
};
