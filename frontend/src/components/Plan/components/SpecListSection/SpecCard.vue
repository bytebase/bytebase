<template>
  <div
    class="spec px-2 py-1 pr-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-stretch overflow-hidden gap-x-1"
    :class="specClass"
    @click="onClickSpec(spec)"
  >
    <div class="w-full flex items-center gap-2">
      <InstanceV1Name
        :instance="databaseForSpec(plan, spec).instanceEntity"
        :link="false"
        class="text-gray-500 text-sm"
      />
      <span class="truncate">{{
        databaseForSpec(plan, spec).databaseName
      }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { databaseForSpec, usePlanContext } from "../../logic";

const props = defineProps<{
  spec: Plan_Spec;
}>();

const { isCreating, plan, selectedSpec, events } = usePlanContext();
const selected = computed(() => props.spec === selectedSpec.value);

const specClass = computed(() => {
  const classes: string[] = [];
  if (isCreating.value) classes.push("create");
  if (selected.value) classes.push("selected");
  return classes;
});

const onClickSpec = (spec: Plan_Spec) => {
  events.emit("select-spec", { spec });
};
</script>

<style scoped lang="postcss">
.spec.selected {
  @apply border-accent bg-accent bg-opacity-5 shadow;
}
.spec .name {
  @apply whitespace-pre-wrap break-all;
}
</style>
