<template>
  <div>
    <SlowQueryPanel
      v-if="database"
      v-model:filter="filter"
      :filter-types="['time-range']"
      :show-project-column="false"
      :show-environment-column="false"
      :show-instance-column="false"
      :show-database-column="false"
    />
  </div>
</template>

<script lang="ts" setup>
import { shallowRef, watch } from "vue";

import type { Database } from "@/types";
import {
  type SlowQueryFilterParams,
  SlowQueryPanel,
  defaultSlowQueryFilterParams,
} from "@/components/SlowQuery";

const props = defineProps<{
  database: Database;
}>();

const filter = shallowRef<SlowQueryFilterParams>({
  ...defaultSlowQueryFilterParams(),
  environment: props.database.instance.environment,
  instance: props.database.instance,
  database: props.database,
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
