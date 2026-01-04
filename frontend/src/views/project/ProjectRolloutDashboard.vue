<template>
  <div class="w-full">
    <div
      class="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2"
    >
      <div class="w-full flex flex-1 items-center justify-between gap-x-2">
        <AdvancedSearch
          v-model:params="state.params"
          class="flex-1"
          :scope-options="scopeOptions"
        />
        <UpdatedTimeRange
          :params="state.params"
          @update:params="state.params = $event"
        />
      </div>
    </div>

    <div class="relative w-full mt-4 min-h-80">
      <PagedTable
        ref="rolloutPagedTable"
        :key="project.name"
        :session-key="`project-${project.name}-rollouts`"
        :fetch-list="fetchRolloutList"
      >
        <template #table="{ list, loading }">
          <RolloutDataTable
            :bordered="true"
            :loading="loading"
            :rollout-list="list"
          />
        </template>
      </PagedTable>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import AdvancedSearch from "@/components/AdvancedSearch";
import type { ScopeOption } from "@/components/AdvancedSearch/types";
import UpdatedTimeRange from "@/components/AdvancedSearch/UpdatedTimeRange.vue";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import RolloutDataTable from "@/components/Rollout/RolloutDataTable.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useProjectByName, useRolloutStore } from "@/store";
import {
  buildRolloutFindBySearchParams,
  type RolloutFind,
} from "@/store/modules/rollout";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  type Rollout,
  Task_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  type SearchParams,
  type SearchScope,
  type SearchScopeId,
} from "@/utils";

interface LocalState {
  params: SearchParams;
}

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const rolloutStore = useRolloutStore();
const rolloutPagedTable = ref<ComponentExposed<typeof PagedTable<Rollout>>>();

const readonlyScopes = computed((): SearchScope[] => {
  return [];
});

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [...readonlyScopes.value],
  };
  return params;
};

const state = reactive<LocalState>({
  params: defaultSearchParams(),
});

watch(
  () => project.value.name,
  () => (state.params = defaultSearchParams())
);

const supportedScopes = computed(() => {
  const supportedScopes: SearchScopeId[] = ["updated"];
  return supportedScopes;
});

// Custom scope options for rollouts that includes creator functionality
const scopeOptions = computed((): ScopeOption[] => {
  const commonOptions = useCommonSearchScopeOptions(supportedScopes.value);

  return [...commonOptions.value];
});

const rolloutSearchParams = computed(() => {
  const defaultScopes = [
    {
      id: "project",
      value: extractProjectResourceName(project.value.name),
    },
  ];
  return {
    query: state.params.query.trim().toLowerCase(),
    scopes: [...state.params.scopes, ...defaultScopes],
  } as SearchParams;
});

const mergedRolloutFind = computed((): RolloutFind => {
  return buildRolloutFindBySearchParams(rolloutSearchParams.value, {
    taskType: [Task_Type.DATABASE_MIGRATE],
  });
});

const fetchRolloutList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, rollouts } = await rolloutStore.listRollouts({
    find: mergedRolloutFind.value,
    pageSize,
    pageToken,
  });
  return {
    nextPageToken,
    list: rollouts,
  };
};

watch(
  () => JSON.stringify(mergedRolloutFind.value),
  () => rolloutPagedTable.value?.refresh()
);
</script>
