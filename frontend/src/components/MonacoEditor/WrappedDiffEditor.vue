<template>
  <Suspense>
    <DiffEditor v-bind="($attrs as any)" />
    <template #fallback>
      <div ref="spinnerWrapperElRef" :class="classes">
        <BBSpin />
      </div>
    </template>
  </Suspense>
</template>

<script setup lang="ts">
import { useParentElement } from "@vueuse/core";
import { computed, defineAsyncComponent, ref } from "vue";

const DiffEditor = defineAsyncComponent(() => import("./DiffEditor.vue"));

const spinnerWrapperElRef = ref<HTMLElement>();
const parentElRef = useParentElement(spinnerWrapperElRef);

const classes = computed(() => {
  const classes: string[] = [
    "flex",
    "flex-col",
    "items-center",
    "justify-center",
  ];
  const parent = parentElRef.value;
  if (parent) {
    const { position, display } = getComputedStyle(parent);
    if (["relative", "absolute", "fixed"].includes(position)) {
      classes.push("absolute", "inset-0", "bg-white/50");
      return classes;
    }
    if (["flex", "inline-flex"].includes(display)) {
      classes.push("w-full", "h-full", "flex-1");
      return classes;
    }
  }

  classes.push("w-full", "h-full");
  return classes;
});
</script>
