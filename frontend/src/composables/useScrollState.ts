import { useEventListener, useScroll } from "@vueuse/core";
import { computed, ref, Ref, unref, watchEffect } from "vue";
import { MaybeRef } from "@/types";

export const useVerticalScrollState = (
  elemRef: Ref<HTMLElement | undefined>,
  maxHeight: MaybeRef<number>
) => {
  const height = ref(0);
  const updateHeight = () => {
    const elem = elemRef.value;
    if (!elem) {
      height.value = 0;
      return;
    }
    height.value = elem.scrollHeight;
  };
  watchEffect(updateHeight);
  useEventListener("resize", updateHeight);
  const show = computed(() => height.value > unref(maxHeight));

  const scroll = useScroll(elemRef);
  const top = computed(() => {
    return show.value && !scroll.arrivedState.top;
  });
  const bottom = computed(() => {
    return show.value && !scroll.arrivedState.bottom;
  });
  return computed(() => ({
    top: top.value,
    bottom: bottom.value,
  }));
};
