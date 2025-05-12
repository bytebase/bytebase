<template>
  <div class="w-full relative">
    <div
      ref="specBar"
      class="spec-list gap-2 px-4 py-2 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 3xl:grid-cols-5 4xl:grid-cols-6 overflow-y-auto"
      :class="{
        'more-bottom': specBarScrollState.bottom,
        'more-top': specBarScrollState.top,
      }"
      :style="{
        'max-height': `${MAX_LIST_HEIGHT}px`,
      }"
    >
      <SpecCard v-for="(spec, i) in specList" :key="i" :spec="spec" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { useVerticalScrollState } from "@/composables/useScrollState";
import { usePlanContext } from "../../logic";
import SpecCard from "./SpecCard.vue";

const MAX_LIST_HEIGHT = 256;

const { plan } = usePlanContext();
const specBar = ref<HTMLDivElement>();
const specBarScrollState = useVerticalScrollState(specBar, MAX_LIST_HEIGHT);

const specList = computed(() => plan.value.steps.flatMap((step) => step.specs));
</script>

<style scoped lang="postcss">
.spec-list::before {
  @apply absolute top-0 h-4 w-full -ml-4 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.spec-list::after {
  @apply absolute bottom-0 h-4 w-full -ml-4 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.spec-list.more-top::before {
  box-shadow: inset 0 0.3rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
.spec-list.more-bottom::after {
  box-shadow: inset 0 -0.3rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
</style>
