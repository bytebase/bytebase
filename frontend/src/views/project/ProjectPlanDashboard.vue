<template>
  <div v-if="ready" class="w-full">
    <div
      class="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2"
    >
      <div class="w-full flex flex-1 items-center justify-between gap-x-2">
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
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, watch, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useRouter, type LocationQuery } from "vue-router";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { targetsForSpec, getLocalSheetByName } from "@/components/Plan";
import AddSpecDrawer from "@/components/Plan/components/AddSpecDrawer.vue";
import PlanDataTable from "@/components/Plan/components/PlanDataTable";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useStorageStore } from "@/store";
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
}

const { enabledNewLayout } = useIssueLayoutVersion();
const { project, ready } = useCurrentProjectV1();
const showAddSpecDrawer = ref(false);

const readonlyScopes = computed((): SearchScope[] => {
  return [];
});

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [
      ...readonlyScopes.value,
      {
        id: "state",
        value: "ACTIVE",
      },
    ],
  };
  return params;
};

const router = useRouter();
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
  const supportedScopes: SearchScopeId[] = ["state"];
  return supportedScopes;
});

const scopeOptions = useCommonSearchScopeOptions(supportedScopes.value);

const planSearchParams = computed(() => {
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

const mergedPlanFind = computed(() => {
  const defaultFind = enabledNewLayout.value
    ? {}
    : // Default find for legacy layout.
      {
        hasIssue: false,
        hasPipeline: false,
      };
  return buildPlanFindBySearchParams(planSearchParams.value, {
    // Only show change database plans.
    specType: "change_database_config",
    ...defaultFind,
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
    spec.config?.case === "changeDatabaseConfig" &&
    spec.config.value.type === Plan_ChangeDatabaseConfig_Type.DATA
      ? "bb.issue.database.data.update"
      : "bb.issue.database.schema.update";
  const targets = targetsForSpec(spec);
  const isDatabaseGroup = targets.every((target) =>
    isValidDatabaseGroupName(target)
  );
  const query: LocationQuery = {
    template,
  };

  // Check if the spec has a sheet with content
  if (spec.config?.case === "changeDatabaseConfig" && spec.config.value.sheet) {
    const sheet = getLocalSheetByName(spec.config.value.sheet);
    if (sheet.content && sheet.content.length > 0) {
      // Convert Uint8Array to string
      const statement = new TextDecoder().decode(sheet.content);
      if (statement) {
        // For single database, use sqlStorageKey
        if (!isDatabaseGroup && targets.length === 1) {
          const sqlStorageKey = `bb.plans.sql.${uuidv4()}`;
          useStorageStore().put(sqlStorageKey, statement);
          query.sqlStorageKey = sqlStorageKey;
        } else {
          // For multiple databases or database groups, use sqlMapStorageKey
          const sqlMap: Record<string, string> = {};
          targets.forEach((target) => {
            sqlMap[target] = statement;
          });
          const sqlMapStorageKey = `bb.plans.sql-map.${uuidv4()}`;
          useStorageStore().put(sqlMapStorageKey, sqlMap);
          query.sqlMapStorageKey = sqlMapStorageKey;
        }
      }
    }
  }

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
