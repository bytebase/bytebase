<template>
  <slot />
</template>

<script lang="ts">
import { useEventListener } from "@vueuse/core";
import Emittery from "emittery";
import { last, pullAt } from "lodash-es";
import { unref } from "vue";
import {
  computed,
  defineComponent,
  inject,
  onUnmounted,
  provide,
  Ref,
  ref,
  watch,
} from "vue";
import { MaybeRef } from "@/types";

export type OverlayStackEvents = Emittery<{
  esc: KeyboardEvent;
}>;

export type OverlayStackContext = {
  stack: Ref<number[]>;
  serial: number;
  events: OverlayStackEvents;
};

const CONTEXT_KEY = Symbol("bb.overlay-stack-manager");

export function useOverlayStackContext() {
  const context = inject<OverlayStackContext>(CONTEXT_KEY)!;
  return context;
}

export function useOverlayStack(show: MaybeRef<boolean> = true) {
  const context = inject<OverlayStackContext>(CONTEXT_KEY)!;
  const { stack, events } = context;

  const id = context.serial++;
  const index = computed(() => stack.value.indexOf(id));
  const upmost = computed(() => id === last(stack.value));

  const push = () => {
    const index = stack.value.indexOf(id);
    if (index >= 0) {
      pullAt(stack.value, index);
    }
    stack.value.push(id);
  };
  const pop = () => {
    const index = stack.value.indexOf(id);
    if (index >= 0) {
      pullAt(stack.value, index);
    }
  };

  useEventListener(
    "keyup",
    (e) => {
      const el = document.activeElement;
      if (!el) return;
      if (e.defaultPrevented) return;

      const EXCLUDED_SELECTORS = [
        "input",
        "textarea",
        ".n-input",
        ".n-base-selection-input",
        ".n-base-selection-tags",
        ".n-base-selection-label",
      ];
      const selector = EXCLUDED_SELECTORS.join(",");
      if (el.matches(selector)) return;

      if (e.key === "Escape" && upmost.value) {
        e.stopPropagation();
        e.stopImmediatePropagation();
        e.preventDefault();
        console.debug(
          "<OverlayStackManager> esc handled, if you don't want to handle esc event when focusing this kind of element, please check the element",
          el
        );

        context.events.emit("esc", e);
      }
    },
    {
      capture: true,
    }
  );

  watch(
    () => unref(show),
    (show) => {
      if (show) push();
      else pop();
    },
    { immediate: true }
  );

  onUnmounted(() => {
    pop();
  });

  return { stack, id, index, upmost, push, pop, events };
}

export default defineComponent({
  name: "BBOverlayStack",
  setup() {
    const context: OverlayStackContext = {
      stack: ref([]),
      serial: 0,
      events: new Emittery(),
    };

    provide<OverlayStackContext>(CONTEXT_KEY, context);
  },
});
</script>
