<template>
  <div class="p-2">
    <SlowQueryPanel v-model:filter="filter" />
  </div>
</template>

<script lang="ts" setup>
import { shallowRef } from "vue";

import {
  SlowQueryPanel,
  SlowQueryFilterParams,
  defaultSlowQueryFilterParams,
} from "@/components/SlowQuery";

const filter = shallowRef<SlowQueryFilterParams>({
  ...defaultSlowQueryFilterParams(),
});
</script>
