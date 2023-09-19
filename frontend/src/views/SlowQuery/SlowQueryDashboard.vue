<template>
  <div class="p-2">
    <SlowQueryPanel v-if="ready" v-model:filter="filter" />
  </div>
</template>

<script lang="ts" setup>
import { shallowRef, watch, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  SlowQueryPanel,
  SlowQueryFilterParams,
  defaultSlowQueryFilterParams,
} from "@/components/SlowQuery";
import {
  extractSlowQueryLogFilterFromQuery,
  wrapQueryFromFilterParams,
} from "./utils";

const route = useRoute();
const router = useRouter();
const ready = shallowRef(false);
const filter = shallowRef<SlowQueryFilterParams>({
  ...defaultSlowQueryFilterParams(),
});

const syncFilterParamsFromQuery = async () => {
  const params = await extractSlowQueryLogFilterFromQuery(route.query);
  filter.value = params;
  ready.value = true;
};

watchEffect(syncFilterParamsFromQuery);

watch(
  filter,
  () => {
    const query = wrapQueryFromFilterParams(filter.value);
    router.replace({
      ...route,
      query,
    });
  },
  { deep: true }
);
</script>
