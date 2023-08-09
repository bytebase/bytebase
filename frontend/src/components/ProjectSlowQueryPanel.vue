<template>
  <div>
    <SlowQueryPanel
      v-model:filter="filter"
      :filter-types="['environment', 'instance', 'database', 'time-range']"
      :show-project-column="false"
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
import { Project } from "@/types/proto/v1/project_service";

const props = defineProps<{
  project: Project;
}>();

const filter = shallowRef<SlowQueryFilterParams>({
  ...defaultSlowQueryFilterParams(),
  project: props.project,
});

watch(
  () => props.project.uid,
  () => {
    filter.value.project = props.project;
  }
);
</script>
