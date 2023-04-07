<template>
  <div class="p-2">
    <SlowQueryPanel v-model:filter="filter" />
  </div>
</template>

<script lang="ts" setup>
import { shallowRef } from "vue";

import { SlowQueryPanel, SlowQueryFilterParams } from "@/components/SlowQuery";

const filter = shallowRef<SlowQueryFilterParams>({
  project: undefined,
  environment: undefined,
  instance: undefined,
  database: undefined,
  timeRange: undefined,
});
</script>
