<template>
  <div>
    <div v-if="filterTypes.length > 0">
      <LogFilter
        :params="filter"
        :filter-types="filterTypes"
        :loading="loading"
        @update:params="$emit('update:filter', $event)"
      >
        <template #suffix>
          <NButton type="default" :loading="syncing" @click="syncNow">
            {{ $t("common.sync-now") }}
          </NButton>
        </template>
      </LogFilter>
    </div>
    <div class="relative min-h-[8rem]">
      <LogTable
        :slow-query-log-list="slowQueryLogList"
        :show-placeholder="!loading"
        :show-project-column="showProjectColumn"
        :show-environment-column="showEnvironmentColumn"
        :show-instance-column="showInstanceColumn"
        :show-database-column="showDatabaseColumn"
        @select="selectSlowQueryLog"
      />
      <div
        v-if="loading"
        class="absolute inset-0 bg-white/50 flex flex-col items-center pt-[6rem]"
      >
        <BBSpin />
      </div>
    </div>

    <DetailPanel
      :slow-query-log="selectedSlowQueryLog"
      @close="selectedSlowQueryLog = undefined"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, shallowRef, watch } from "vue";
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";

import { ComposedSlowQueryLog } from "@/types";
import {
  pushNotification,
  useGracefulRequest,
  useSlowQueryPolicyStore,
  useSlowQueryStore,
} from "@/store";
import {
  type FilterType,
  type SlowQueryFilterParams,
  FilterTypeList,
  buildListSlowQueriesRequest,
} from "./types";
import LogFilter from "./LogFilter.vue";
import LogTable from "./LogTable.vue";
import DetailPanel from "./DetailPanel.vue";
import { extractInstanceResourceName } from "@/utils";

const props = withDefaults(
  defineProps<{
    filter: SlowQueryFilterParams;
    filterTypes?: readonly FilterType[];
    showProjectColumn?: boolean;
    showEnvironmentColumn?: boolean;
    showInstanceColumn?: boolean;
    showDatabaseColumn?: boolean;
  }>(),
  {
    filterTypes: () => FilterTypeList,
    showProjectColumn: true,
    showEnvironmentColumn: true,
    showInstanceColumn: true,
    showDatabaseColumn: true,
  }
);

defineEmits<{
  (event: "update:filter", filter: SlowQueryFilterParams): void;
}>();

const { t } = useI18n();
const slowQueryStore = useSlowQueryStore();
const loading = shallowRef(false);
const slowQueryLogList = shallowRef<ComposedSlowQueryLog[]>([]);
const selectedSlowQueryLog = shallowRef<ComposedSlowQueryLog>();
const syncing = shallowRef(false);

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

const selectSlowQueryLog = (log: ComposedSlowQueryLog) => {
  selectedSlowQueryLog.value = log;
};

const syncNow = async () => {
  syncing.value = true;
  try {
    await useGracefulRequest(async () => {
      const policyList = await useSlowQueryPolicyStore().fetchPolicyList();
      const requestList = policyList
        .filter((policy) => {
          return policy.slowQueryPolicy?.active;
        })
        .map(async (policy) => {
          return slowQueryStore.syncSlowQueriesByInstance(
            `instances/${extractInstanceResourceName(policy.name)}`
          );
        });
      await Promise.all(requestList);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("slow-query.sync-job-started"),
      });
    });
  } finally {
    syncing.value = false;
  }
};

// Fetch the list while params changed.
watch(() => JSON.stringify(params.value), fetchSlowQueryLogList, {
  immediate: true,
});
</script>
