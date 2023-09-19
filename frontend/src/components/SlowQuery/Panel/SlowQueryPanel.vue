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
          <NButton v-if="allowAdmin" type="default" @click="goConfig">
            {{ $t("common.configure") }}
          </NButton>
          <NButton
            type="default"
            :disabled="!allowSync"
            :loading="syncing"
            @click="syncNow"
          >
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
        :allow-admin="allowAdmin"
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
import { NButton } from "naive-ui";
import { computed, shallowRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  pushNotification,
  useCurrentUserV1,
  useGracefulRequest,
  useSlowQueryPolicyList,
  useSlowQueryPolicyStore,
  useSlowQueryStore,
} from "@/store";
import { ComposedSlowQueryLog } from "@/types";
import {
  extractInstanceResourceName,
  extractProjectResourceName,
  hasWorkspacePermissionV1,
} from "@/utils";
import DetailPanel from "./DetailPanel.vue";
import LogFilter from "./LogFilter.vue";
import LogTable from "./LogTable.vue";
import {
  type FilterType,
  type SlowQueryFilterParams,
  FilterTypeList,
  buildListSlowQueriesRequest,
} from "./types";

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
const router = useRouter();
const currentUser = useCurrentUserV1();
const slowQueryStore = useSlowQueryStore();
const loading = shallowRef(false);
const slowQueryLogList = shallowRef<ComposedSlowQueryLog[]>([]);
const selectedSlowQueryLog = shallowRef<ComposedSlowQueryLog>();
const syncing = shallowRef(false);

const params = computed(() => {
  return buildListSlowQueriesRequest(props.filter);
});

const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-slow-query",
    currentUser.value.userRole
  );
});

const { list: slowQueryPolicyList } = useSlowQueryPolicyList();

const allowSync = computed(() => {
  return (
    slowQueryPolicyList.value.filter((policy) => policy.slowQueryPolicy?.active)
      .length > 0
  );
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
      if (props.filter.instance) {
        await slowQueryStore.syncSlowQueries(
          `instances/${extractInstanceResourceName(props.filter.instance.name)}`
        );
      } else if (props.filter.project) {
        await slowQueryStore.syncSlowQueries(
          `projects/${extractProjectResourceName(props.filter.project.name)}`
        );
      } else {
        const policyList = await useSlowQueryPolicyStore().fetchPolicyList();
        const requestList = policyList
          .filter((policy) => {
            return policy.slowQueryPolicy?.active;
          })
          .map(async (policy) => {
            return slowQueryStore.syncSlowQueries(
              `instances/${extractInstanceResourceName(policy.name)}`
            );
          });
        await Promise.all(requestList);
      }

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

const goConfig = () => {
  router.push({
    name: "setting.workspace.slow-query",
  });
};

// Fetch the list while params changed.
watch(() => JSON.stringify(params.value), fetchSlowQueryLogList, {
  immediate: true,
});
</script>
