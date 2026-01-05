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
        <PermissionGuardWrapper
          v-slot="slotProps"
          :project="project"
          :permissions="['bb.plans.create']"
        >
          <NButton
            type="primary"
            :disabled="slotProps.disabled"
            @click="showAddSpecDrawer = true"
          >
            <template #icon>
              <PlusIcon class="w-4 h-4" />
            </template>
            {{ $t("plan.new-plan") }}
          </NButton>
        </PermissionGuardWrapper>
      </div>
    </div>

    <div class="relative w-full mt-4 min-h-80">
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
import { computed, h, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import { type LocationQuery, useRouter } from "vue-router";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import AdvancedSearch from "@/components/AdvancedSearch";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { getLocalSheetByName, targetsForSpec } from "@/components/Plan";
import AddSpecDrawer from "@/components/Plan/components/AddSpecDrawer.vue";
import PlanDataTable from "@/components/Plan/components/PlanDataTable";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import {
  useCurrentProjectV1,
  useCurrentUserV1,
  useStorageStore,
  useUserStore,
} from "@/store";
import {
  buildPlanFindBySearchParams,
  usePlanStore,
} from "@/store/modules/v1/plan";
import { isValidDatabaseGroupName, SYSTEM_BOT_USER_NAME } from "@/types";
import { type Plan, type Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractProjectResourceName,
  generateIssueTitle,
  getDefaultPagination,
  type SearchParams,
  type SearchScope,
  type SearchScopeId,
} from "@/utils";

interface LocalState {
  params: SearchParams;
}

const { t } = useI18n();
const me = useCurrentUserV1();
const { enabledNewLayout } = useIssueLayoutVersion();
const { project, ready } = useCurrentProjectV1();
const userStore = useUserStore();
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
  const supportedScopes: SearchScopeId[] = ["state", "creator"];
  return supportedScopes;
});

// Custom scope options for plans that includes creator functionality
const scopeOptions = computed((): ScopeOption[] => {
  // Get common options but exclude both creator and state (we'll add them manually)
  const commonOptions = useCommonSearchScopeOptions(
    supportedScopes.value.filter((id) => id !== "creator" && id !== "state")
  );

  const renderSpan = (text: string) => h("span", text);

  const searchPrincipalSearchValueOptions = (userTypes: UserType[]) => {
    return ({
      keyword,
      nextPageToken,
    }: {
      keyword: string;
      nextPageToken?: string;
    }) =>
      userStore
        .fetchUserList({
          pageToken: nextPageToken,
          pageSize: getDefaultPagination(),
          filter: {
            types: userTypes,
            query: keyword,
          },
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken,
          options: resp.users.map<ValueOption>((user) => {
            return {
              value: user.email,
              keywords: [user.email, user.title],
              bot: user.name === SYSTEM_BOT_USER_NAME,
              render: () => {
                const children = [
                  h(BBAvatar, { size: "TINY", username: user.title }),
                  renderSpan(user.title),
                ];
                if (user.name === me.value.name) {
                  children.push(h(YouTag));
                }
                if (user.name === SYSTEM_BOT_USER_NAME) {
                  children.push(h(SystemBotTag));
                }
                return h(
                  "div",
                  { class: "flex items-center gap-x-1" },
                  children
                );
              },
            };
          }),
        }));
  };

  const creatorOption: ScopeOption = {
    id: "creator",
    title: t("issue.advanced-search.scope.creator.title"),
    description: t("issue.advanced-search.scope.creator.description"),
    search: searchPrincipalSearchValueOptions([
      UserType.USER,
      UserType.SERVICE_ACCOUNT,
      UserType.SYSTEM_BOT,
    ]),
  };

  const stateOption: ScopeOption = {
    id: "state",
    title: t("common.state"),
    description: t("plan.state.description"),
    options: [
      {
        value: "ACTIVE",
        keywords: ["active", "ACTIVE"],
        render: () => renderSpan(t("common.active")),
      },
      {
        value: "DELETED",
        keywords: ["deleted", "DELETED", "closed", "CLOSED"],
        render: () => renderSpan(t("common.closed")),
      },
    ],
    allowMultiple: false,
  };

  return [...commonOptions.value, creatorOption, stateOption];
});

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
        hasRollout: false,
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
  const { nextPageToken, plans } = await planStore.listPlans({
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

const handleSpecCreated = async (spec: Plan_Spec) => {
  const template = "bb.issue.database.update";
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
    // Only set title from generated if enforceIssueTitle is false.
    if (!project.value.enforceIssueTitle) {
      query.name = generateIssueTitle(template, [
        extractDatabaseGroupName(databaseGroupName),
      ]);
    }
  } else {
    query.databaseList = targets.join(",");
    // Only set title from generated if enforceIssueTitle is false.
    if (!project.value.enforceIssueTitle) {
      query.name = generateIssueTitle(
        template,
        targets.map((db) => {
          const { databaseName } = extractDatabaseResourceName(db);
          return databaseName;
        })
      );
    }
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
