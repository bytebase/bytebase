<template>
  <teleport to="body">
    <!-- use v-show, not v-if. we need to keep this element in dom tree -->
    <!-- because it is used as teleport target -->
    <!-- removing it from tree may cause teleport error-->
    <div
      v-show="stack.length > 0"
      id="bb-modal-stack"
      v-zindexable="{ enabled: true }"
      class="fixed inset-0 w-full h-screen flex items-center justify-center bg-gray-300/80"
      :data-bb-modal-stack="stack.join('|')"
    />
  </teleport>
  <slot />
</template>

<script lang="ts">
import { zindexable } from "vdirs";
import {
  computed,
  defineComponent,
  inject,
  onUnmounted,
  provide,
  Ref,
  ref,
} from "vue";

export type ModalStackContext = {
  stack: Ref<number[]>;
  autoIncrement: number;
};

const CONTEXT_KEY = "bb.modal-stack";

export function useModalStack() {
  const context = inject<ModalStackContext>(CONTEXT_KEY)!;
  const { stack } = context;

  const id = context.autoIncrement++;
  stack.value.push(id);
  const index = computed(() => stack.value.indexOf(id));
  const active = computed(() => index.value === stack.value.length - 1);

  onUnmounted(() => {
    const index = stack.value.indexOf(id);
    if (index >= 0) {
      stack.value.splice(index, 1);
    }
  });

  return { stack, id, index, active };
}

export function useModalStackStatus() {
  const { stack } = inject<ModalStackContext>(CONTEXT_KEY)!;
  return stack;
}

export default defineComponent({
  name: "BBModalStack",
  directives: {
    zindexable,
  },
  setup() {
    const stack = ref<number[]>([]);

    provide<ModalStackContext>(CONTEXT_KEY, {
      stack,
      autoIncrement: 0,
    });

    return {
      stack,
    };
  },
});
</script>
