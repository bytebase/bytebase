<template>
  <div :class="['w-full', !specificProject && 'px-4']">
    <div
      class="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2"
    >
      <div class="flex flex-1 max-w-full items-center gap-x-2">
        <AdvancedSearch
          v-model:params="state.params"
          class="flex-1"
          :scope-options="scopeOptions"
        />
        <NDropdown
          trigger="hover"
          :options="planCreationButtonOptions"
          :disabled="!allowToCreatePlan"
          @select="state.selectedPlanOnlyType = $event"
        >
          <NButton type="primary" :disabled="!allowToCreatePlan">{{
            $t("review-center.review-sql")
          }}</NButton>
        </NDropdown>
      </div>
    </div>

    <div class="relative w-full mt-4 min-h-[20rem]">
      <PagedTable
        ref="planPagedTable"
        :session-key="'review-center'"
        :fetch-list="fetchPlanList"
      >
        <template #table="{ list, loading }">
          <PlanDataTable
            :loading="loading"
            :plan-list="list"
            :show-project="!specificProject"
          />
        </template>
      </PagedTable>
    </div>
  </div>

  <Drawer
    :auto-focus="true"
    :show="state.selectedPlanOnlyType !== undefined"
    @close="state.selectedPlanOnlyType = undefined"
  >
    <AlterSchemaPrepForm
      :project-name="specificProject.name"
      :type="state.selectedPlanOnlyType!"
      :plan-only="true"
      @dismiss="state.selectedPlanOnlyType = undefined"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton, NDropdown, type DropdownOption } from "naive-ui";
import { computed, reactive, watch, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import AlterSchemaPrepForm from "@/components/AlterSchemaPrepForm/";
import PlanDataTable from "@/components/Plan/components/PlanDataTable";
import { Drawer } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import {
  useCurrentUserV1,
  useProjectByName,
  useRefreshPlanList,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { usePlanStore } from "@/store/modules/v1/plan";
import { buildPlanFindBySearchParams } from "@/store/modules/v1/plan";
import type { ComposedPlan } from "@/types/v1/issue/plan";
import {
  extractProjectResourceName,
  hasPermissionToCreatePlanInProject,
  type SearchParams,
  type SearchScope,
  type SearchScopeId,
} from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

interface LocalState {
  params: SearchParams;
  selectedPlanOnlyType?:
    | "bb.issue.database.schema.update"
    | "bb.issue.database.data.update";
}

const { project: specificProject } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const readonlyScopes = computed((): SearchScope[] => {
  return [
    {
      id: "project",
      value: extractProjectResourceName(specificProject.value.name),
      readonly: true,
    },
  ];
});

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [...readonlyScopes.value],
  };
  return params;
};

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const state = reactive<LocalState>({
  params: defaultSearchParams(),
});

watch(
  () => specificProject.value,
  () => (state.params = defaultSearchParams())
);

const planStore = usePlanStore();
const planPagedTable = ref<ComponentExposed<typeof PagedTable<ComposedPlan>>>();

const planCreationButtonOptions = computed((): DropdownOption[] => {
  return [
    {
      label: `${t("database.edit-schema")} (DDL)`,
      key: "bb.issue.database.schema.update",
    },
    {
      label: `${t("database.change-data")} (DML)`,
      key: "bb.issue.database.data.update",
    },
  ];
});

const supportedScopes = computed(() => {
  // TODO(steven): support more scopes in backend and frontend: instance, database, planCheckRunStatus
  const supportedScopes: SearchScopeId[] = [];
  return supportedScopes;
});

const scopeOptions = useCommonSearchScopeOptions(supportedScopes.value);

const planSearchParams = computed(() => {
  // Default scopes with type and creator.
  const defaultScopes = [
    {
      id: "creator",
      value: currentUser.value.email,
    },
  ];
  // If specific project is provided, add project scope.
  if (specificProject.value) {
    defaultScopes.push({
      id: "project",
      value: extractProjectResourceName(specificProject.value.name),
    });
  }
  return {
    scopes: [...state.params.scopes, ...defaultScopes],
  } as SearchParams;
});

const mergedPlanFind = computed(() => {
  return buildPlanFindBySearchParams(planSearchParams.value, {
    hasIssue: false,
    hasPipeline: false,
  });
});

const fetchPlanList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, plans } = await planStore.searchPlans({
    find: mergedPlanFind.value,
    pageSize,
    pageToken,
  });
  return {
    nextPageToken,
    list: plans,
  };
};

watch(
  () => JSON.stringify(mergedPlanFind.value),
  () => planPagedTable.value?.refresh()
);
useRefreshPlanList(() => planPagedTable.value?.refresh());

const allowToCreatePlan = computed(() => {
  // Check if user has permission to create plan in specific project.
  return hasPermissionToCreatePlanInProject(specificProject.value);
});
</script>
