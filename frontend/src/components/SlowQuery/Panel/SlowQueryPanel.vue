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
import dayjs from "dayjs";

import { useSlowQueryStore } from "@/store";
import type {
  ListSlowQueriesRequest,
  SlowQueryLog,
} from "@/types/proto/v1/database_service";
import { UNKNOWN_ID } from "@/types";
import {
  type FilterType,
  type SlowQueryFilterParams,
  FilterTypeList,
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
const slowQueryLogList = shallowRef<SlowQueryLog[]>([]);
const selectedSlowQueryLog = shallowRef<SlowQueryLog>();

const params = computed(() => {
  const request = {} as Partial<ListSlowQueriesRequest>;
  const query: string[] = [];
  const { filter } = props;
  const { project, environment, instance, database, timeRange } = filter;

  if (database && database.id !== UNKNOWN_ID) {
    request.parent = `environments/${database.instance.environment.resourceId}/instances/${database.instance.resourceId}/databases/${database.id}`;
  } else if (instance && instance.id !== UNKNOWN_ID) {
    request.parent = `environments/${instance.environment.resourceId}/instances/${instance.resourceId}/databases/-`;
  } else if (environment && environment.id !== UNKNOWN_ID) {
    request.parent = `environments/${environment.resourceId}/instances/-/databases/-`;
  }

  if (project) {
    query.push(`project = projects/${project.resourceId}`);
  }
  if (timeRange) {
    const start = dayjs(timeRange[0]).startOf("day").toISOString();
    const end = dayjs(timeRange[1]).endOf("day").toISOString();
    query.push(`start_time >= ${start}`);
    query.push(`start_time <= ${end}`);
  }
  if (query.length > 0) {
    request.filter = query.join(" && ");
  }
  return request;
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
