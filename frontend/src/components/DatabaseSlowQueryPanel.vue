<template>
  <div>
    <SlowQueryPanel
      v-if="database"
      v-model:filter="filter"
      :filter-types="['time-range']"
    />
  </div>
</template>

<script lang="ts" setup>
import { shallowRef, watch } from "vue";

import type { Database } from "@/types";
import {
  type SlowQueryFilterParams,
  SlowQueryPanel,
} from "@/components/SlowQuery";

const props = defineProps<{
  database: Database;
}>();

const filter = shallowRef<SlowQueryFilterParams>({
  project: undefined,
  environment: props.database.instance.environment,
  instance: props.database.instance,
  database: props.database,
  timeRange: undefined,
});

watch(
  () => props.database.id,
  () => {
    filter.value.environment = props.database.instance.environment;
    filter.value.instance = props.database.instance;
    filter.value.database = props.database;
  }
);
</script>
