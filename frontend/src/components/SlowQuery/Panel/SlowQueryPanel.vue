<template>
  <div>
    <div v-if="filterTypes.length > 0">
      <LogFilter
        :params="filter"
        :filter-types="filterTypes"
        :loading="loading"
        @update:params="$emit('update:filter', $event)"
      />
    </div>
    <div class="relative min-h-[8rem]">
      <LogTable
        :slow-query-log-list="slowQueryLogList"
        :show-placeholder="!loading"
        @select="selectedSlowQueryLog = $event"
      />
      <div
        v-if="loading"
        class="absolute inset-0 bg-white/50 flex flex-col items-center pt-[6rem]"
      >
        <BBSpin />
      </div>
    </div>

    <DetailDialog
      v-if="selectedSlowQueryLog"
      :slow-query-log="selectedSlowQueryLog"
      @close="selectedSlowQueryLog = undefined"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, shallowRef, watch } from "vue";

import type { ComposedSlowQueryLog } from "@/types";
import { useSlowQueryStore } from "@/store";
import {
  type FilterType,
  type SlowQueryFilterParams,
  FilterTypeList,
  buildListSlowQueriesRequest,
} from "./types";
import LogFilter from "./LogFilter.vue";
import LogTable from "./LogTable.vue";
import DetailDialog from "./DetailDialog.vue";

const props = withDefaults(
  defineProps<{
    filter: SlowQueryFilterParams;
    filterTypes?: readonly FilterType[];
  }>(),
  {
    filterTypes: () => FilterTypeList,
  }
);

defineEmits<{
  (event: "update:filter", filter: SlowQueryFilterParams): void;
}>();

const slowQueryStore = useSlowQueryStore();
const loading = shallowRef(false);
const slowQueryLogList = shallowRef<ComposedSlowQueryLog[]>([]);
const selectedSlowQueryLog = shallowRef<ComposedSlowQueryLog>();

const params = computed(() => {
  return buildListSlowQueriesRequest(props.filter);
});

const fetchSlowQueryLogList = async () => {
  loading.value = true;
  try {
    const list = await slowQueryStore.fetchSlowQueryLogList(params.value);
    slowQueryLogList.value = list;
  } finally {
    loading.value = false;
  }
};

// Fetch the list while params changed.
watch(() => JSON.stringify(params.value), fetchSlowQueryLogList, {
  immediate: true,
});
</script>
