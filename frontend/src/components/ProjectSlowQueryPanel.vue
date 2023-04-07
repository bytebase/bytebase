<template>
  <div>
    <SlowQueryPanel
      v-model:filter="filter"
      :filter-types="['environment', 'database', 'time-range']"
    />
  </div>
</template>

<script lang="ts" setup>
import { shallowRef, watch } from "vue";

import { SlowQueryPanel, SlowQueryFilterParams } from "@/components/SlowQuery";
import { Project } from "@/types";

const props = defineProps<{
  project: Project;
}>();

const filter = shallowRef<SlowQueryFilterParams>({
  project: props.project,
  environment: undefined,
  instance: undefined,
  database: undefined,
  timeRange: undefined,
});

watch(
  () => props.project.id,
  () => {
    filter.value.project = props.project;
  }
);
</script>
