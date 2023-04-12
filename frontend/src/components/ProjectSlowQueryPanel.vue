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

import {
  SlowQueryPanel,
  SlowQueryFilterParams,
  defaultSlowQueryFilterParams,
} from "@/components/SlowQuery";
import { Project } from "@/types";

const props = defineProps<{
  project: Project;
}>();

const filter = shallowRef<SlowQueryFilterParams>({
  ...defaultSlowQueryFilterParams(),
  project: props.project,
});

watch(
  () => props.project.id,
  () => {
    filter.value.project = props.project;
  }
);
</script>
