<template>
  <teleport to="body">
    <!-- use v-show, not v-if. we need to keep this element in dom tree -->
    <!-- because it is used as teleport target -->
    <!-- removing it from tree may cause teleport error-->
    <div
      v-show="stack.length > 0"
      id="bb-modal-stack"
      class="bb-modal-mask"
      :data-bb-modal-stack="stack.join('|')"
    />
  </teleport>
  <slot />
</template>

<script lang="ts">
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

<style scope>
.bb-modal-mask {
  @apply fixed inset-0 w-full h-screen flex items-center justify-center z-[4000];
  background-color: rgba(209, 213, 219, 0.8); /* bg-gray-300 */
}
</style>
