<template>
  <div v-if="ready" class="w-full">
    <div
      class="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2"
    >
      <div class="flex flex-1 max-w-full items-center gap-x-2">
        <AdvancedSearch
          v-model:params="state.params"
          class="flex-1"
          :scope-options="scopeOptions"
        />
        <NButton
          v-if="allowToCreatePlan"
          type="primary"
          @click="showAddSpecDrawer = true"
        >
          <template #icon>
            <PlusIcon class="w-4 h-4" />
          </template>
          {{ $t("plan.new-plan") }}
        </NButton>
      </div>
    </div>

    <div class="relative w-full mt-4 min-h-[20rem]">
      <PagedTable
        ref="planPagedTable"
        :session-key="`bb.${project.name}.plan-table`"
        :fetch-list="fetchPlanList"
      >
        <template #table="{ list, loading }">
          <PlanDataTable :loading="loading" :plan-list="list" />
        </template>
      </PagedTable>
    </div>
  </div>

  <AddSpecDrawer
    v-model:show="showAddSpecDrawer"
    :title="$t('plan.new-plan')"
    @created="handleSpecCreated"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive, watch, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useRouter } from "vue-router";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { targetsForSpec } from "@/components/Plan";
import AddSpecDrawer from "@/components/Plan/components/AddSpecDrawer.vue";
import PlanDataTable from "@/components/Plan/components/PlanDataTable";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useCurrentUserV1 } from "@/store";
import { usePlanStore } from "@/store/modules/v1/plan";
import { buildPlanFindBySearchParams } from "@/store/modules/v1/plan";
import { isValidDatabaseGroupName } from "@/types";
import {
  Plan_ChangeDatabaseConfig_Type,
  type Plan,
  type Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractProjectResourceName,
  generateIssueTitle,
  hasPermissionToCreatePlanInProject,
  type SearchParams,
  type SearchScope,
  type SearchScopeId,
} from "@/utils";

interface LocalState {
  params: SearchParams;
  selectedPlanOnlyType?:
    | "bb.issue.database.schema.update"
    | "bb.issue.database.data.update";
}

const { project, ready } = useCurrentProjectV1();
const showAddSpecDrawer = ref(false);

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

const router = useRouter();
const currentUser = useCurrentUserV1();
const state = reactive<LocalState>({
  params: defaultSearchParams(),
});

watch(
  () => project.value.name,
  () => (state.params = defaultSearchParams())
);

const planStore = usePlanStore();
const planPagedTable = ref<ComponentExposed<typeof PagedTable<Plan>>>();

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
    {
      id: "project",
      value: extractProjectResourceName(project.value.name),
    },
  ];
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

const allowToCreatePlan = computed(() => {
  // Check if user has permission to create plan in specific project.
  return hasPermissionToCreatePlanInProject(project.value);
});

const handleSpecCreated = async (spec: Plan_Spec) => {
  const template =
    spec.config?.case === "changeDatabaseConfig" && spec.config.value.type === Plan_ChangeDatabaseConfig_Type.DATA
      ? "bb.issue.database.data.update"
      : "bb.issue.database.schema.update";
  const targets = targetsForSpec(spec);
  const isDatabaseGroup = targets.every((target) =>
    isValidDatabaseGroupName(target)
  );
  const query: Record<string, any> = {
    template,
  };
  if (isDatabaseGroup) {
    const databaseGroupName = head(targets);
    if (!databaseGroupName) {
      throw new Error("No valid database group name found in targets.");
    }
    query.databaseGroupName = databaseGroupName;
    query.name = generateIssueTitle(template, [
      extractDatabaseGroupName(databaseGroupName),
    ]);
  } else {
    query.databaseList = targets.join(",");
    query.name = generateIssueTitle(
      template,
      targets.map((db) => {
        const { databaseName } = extractDatabaseResourceName(db);
        return databaseName;
      })
    );
  }

  // Navigate to the spec detail page with the created spec.
  router.push({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      planId: "create",
      specId: "placeholder", // This will be replaced with the actual spec ID later.
    },
    query,
  });
};
</script>
