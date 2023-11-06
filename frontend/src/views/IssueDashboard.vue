<template>
  <div class="flex flex-col">
    <AdvancedSearch
      custom-class="w-full px-4 py-2"
      :params="initSearchParams"
      :autofocus="autofocus"
      @update="onSearchParamsUpdate($event)"
    />

    <FeatureAttention
      v-if="!!state.searchParams.query || state.searchParams.scopes.length > 0"
      custom-class="w-full px-4 py-2"
      feature="bb.feature.issue-advanced-search"
    />

    <div class="px-4 flex items-center">
      <div class="flex-1 overflow-hidden">
        <TabFilter v-model:value="state.tab" :items="tabItemList" />
      </div>
      <div class="flex flex-row space-x-4 p-0.5">
        <NButton v-if="project" @click="goProject">
          {{ project.key }}
        </NButton>

        <NInputGroup style="width: auto">
          <NDatePicker
            v-model:value="selectedTimeRange"
            type="daterange"
            size="medium"
            :on-confirm="confirmDatePicker"
            :on-clear="clearDatePicker"
            :is-date-disabled="isDateDisabled"
            clearable
          >
          </NDatePicker>
          <SearchBox
            :value="state.filterText"
            :placeholder="$t('common.filter-by-name')"
            :autofocus="false"
            @update:value="state.filterText = $event"
          />
        </NInputGroup>
      </div>
    </div>

    <div v-show="state.tab === 'OPEN'" class="mt-2">
      <!-- show all OPEN issues with pageSize=10  -->
      <PagedIssueTableV1
        session-key="dashboard-open"
        method="SEARCH"
        :issue-filter="{
          ...issueFilter,
          statusList: [IssueStatus.OPEN],
        }"
        :page-size="50"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList.filter(filter)"
            :highlight-text="state.searchParams.query"
            title=""
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="state.tab === 'CLOSED'" class="mt-2">
      <!-- show all DONE and CANCELED issues with pageSize=10 -->
      <PagedIssueTableV1
        session-key="dashboard-closed"
        method="SEARCH"
        :issue-filter="{
          ...issueFilter,
          statusList: [IssueStatus.DONE, IssueStatus.CANCELED],
        }"
        :page-size="50"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList.filter(filter)"
            :highlight-text="state.searchParams.query"
            title=""
          />
        </template>
      </PagedIssueTableV1>
    </div>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NInputGroup, NButton, NDatePicker } from "naive-ui";
import { reactive, computed, watchEffect, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import AdvancedSearch, {
  SearchParams,
  SearchScopeId,
} from "@/components/AdvancedSearch.vue";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { SearchBox, TabFilterItem } from "@/components/v2";
import {
  useCurrentUserV1,
  useProjectV1Store,
  useUserStore,
  useDatabaseV1Store,
  hasFeature,
} from "@/store";
import {
  projectNamePrefix,
  userNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { UNKNOWN_ID, IssueFilter, ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  projectV1Slug,
  extractProjectResourceName,
  hasWorkspacePermissionV1,
} from "@/utils";

const TABS = ["OPEN", "CLOSED"] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  tab: TabValue;
  filterText: string;
  searchParams: SearchParams;
}

const router = useRouter();
const route = useRoute();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();
const databaseV1Store = useDatabaseV1Store();
const userStore = useUserStore();
const currentUserV1 = useCurrentUserV1();

const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-issue",
    currentUserV1.value.userRole
  );
});

const autofocus = computed((): boolean => {
  return !!route.query.autofocus;
});

const initSearchParams = computed((): SearchParams => {
  const projectName = project.value?.name ?? "";
  const userEmail = selectedPrincipal.value?.email ?? "";
  const query = (route.query.query as string) ?? "";

  const params: SearchParams = {
    query,
    scopes: [],
  };

  if (projectName) {
    params.scopes.push({
      id: "project",
      value: extractProjectResourceName(projectName),
    });
  }
  if (userEmail && hasPermission.value) {
    params.scopes.push({
      id: "principal",
      value: userEmail,
    });
  }

  return params;
});

const state = reactive<LocalState>({
  tab: "OPEN",
  filterText: "",
  searchParams: {
    query: "",
    scopes: [],
  },
});
const hasAdvancedSearchFeature = computed(() => {
  return hasFeature("bb.feature.issue-advanced-search");
});

const project = computed(() => {
  if (selectedProjectId.value) {
    return projectV1Store.getProjectByUID(selectedProjectId.value);
  }
  return undefined;
});

const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  const OPEN: TabFilterItem<TabValue> = {
    value: "OPEN",
    label: t("issue.table.open"),
  };
  const CLOSED: TabFilterItem<TabValue> = {
    value: "CLOSED",
    label: t("issue.table.closed"),
  };
  return [OPEN, CLOSED];
});

// timeRangeLimitForFreePlanInTs is the search time limit in ts format.
// should be 60 days.
const timeRangeLimitForFreePlanInTs = 60 * 24 * 60 * 60 * 1000;

const selectedTimeRange = computed((): [number, number] => {
  const today = dayjs().add(1, "day").endOf("day").valueOf();
  const defaultTimeRange = [today - timeRangeLimitForFreePlanInTs, today] as [
    number,
    number
  ];
  const createdTsAfter = route.query.createdTsAfter as string;
  if (createdTsAfter) {
    defaultTimeRange[0] = parseInt(createdTsAfter, 10);
  }
  const createdTsBefore = route.query.createdTsBefore as string;
  if (createdTsBefore) {
    defaultTimeRange[1] = parseInt(createdTsBefore, 10);
  }
  return defaultTimeRange;
});

const isDateDisabled = (ts: number) => {
  const today = dayjs().add(1, "day").endOf("day").valueOf();
  if (ts > today) {
    return true;
  }
  if (hasAdvancedSearchFeature.value) {
    return false;
  }

  return ts < today - timeRangeLimitForFreePlanInTs;
};

const selectedProjectId = computed((): string | undefined => {
  const { project } = route.query;
  return project ? (project as string) : undefined;
});

const selectedPrincipal = computed(() => {
  const { user } = route.query;
  if (!user) {
    return;
  }
  return userStore.getUserById(user as string);
});

const filter = (issue: ComposedIssue) => {
  const keyword = state.filterText.trim().toLowerCase();
  if (keyword) {
    if (!issue.title.toLowerCase().includes(keyword)) {
      return false;
    }
  }
  return true;
};

const confirmDatePicker = (value: [number, number]) => {
  router.replace({
    name: "workspace.issue",
    query: {
      ...route.query,
      createdTsAfter: value[0],
      createdTsBefore: value[1],
    },
  });
};

const clearDatePicker = () => {
  router.replace({
    name: "workspace.issue",
    query: {
      ...route.query,
      createdTsAfter: 0,
      createdTsBefore: Date.now(),
    },
  });
};

const goProject = () => {
  if (!project.value) return;
  router.push({
    name: "workspace.project.detail",
    params: {
      projectSlug: projectV1Slug(project.value),
    },
  });
};

watchEffect(() => {
  if (selectedProjectId.value) {
    projectV1Store.getOrFetchProjectByUID(selectedProjectId.value);
  }
});

onMounted(() => {
  const status = ((route.query.status ?? "") as string).toUpperCase() ?? "OPEN";
  if (status === "CLOSED") {
    state.tab = "CLOSED";
  }
  state.searchParams = initSearchParams.value;
});

const onSearchParamsUpdate = (params: SearchParams) => {
  state.searchParams = params;
};

const getValueFromIssueFilter = (
  prefix: string,
  scopeId: SearchScopeId
): string => {
  const { scopes } = state.searchParams;
  const scope = scopes.find((s) => s.id === scopeId);
  if (!scope) {
    return "";
  }
  return `${prefix}${scope.value}`;
};

const issueFilter = computed((): IssueFilter => {
  const { query, scopes } = state.searchParams;
  const projectScope = scopes.find((s) => s.id === "project");
  const typeScope = scopes.find((s) => s.id === "type");
  const databaseScope = scopes.find((s) => s.id === "database");

  let database = "";
  if (databaseScope) {
    const uid = databaseScope.value.split("-").slice(-1)[0];
    const db = databaseV1Store.getDatabaseByUID(uid);
    if (db.uid !== `${UNKNOWN_ID}`) {
      database = db.name;
    }
  }

  return {
    query,
    instance: getValueFromIssueFilter(instanceNamePrefix, "instance"),
    database,
    project: `${projectNamePrefix}${projectScope?.value ?? "-"}`,
    createdTsAfter: selectedTimeRange.value
      ? selectedTimeRange.value[0]
      : undefined,
    createdTsBefore: selectedTimeRange.value
      ? selectedTimeRange.value[1]
      : undefined,
    type: typeScope?.value,
    principal: getValueFromIssueFilter(userNamePrefix, "principal"),
    creator: getValueFromIssueFilter(userNamePrefix, "creator"),
    assignee: getValueFromIssueFilter(userNamePrefix, "assignee"),
    subscriber: getValueFromIssueFilter(userNamePrefix, "subscriber"),
  };
});
</script>
