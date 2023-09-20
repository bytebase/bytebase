<template>
  <div class="flex flex-col">
    <AdvancedSearch
      custom-class="m-4"
      :params="initSearchParams"
      :autofocus="autofocus"
      @update="onSearchParamsUpdate($event)"
    />

    <FeatureAttention
      v-if="!!state.searchParams.query || state.searchParams.scopes.length > 0"
      custom-class="m-4"
      feature="bb.feature.issue-advanced-search"
    />

    <div class="px-2 flex items-center">
      <div class="flex-1 overflow-hidden">
        <TabFilter v-model:value="state.tab" :items="tabItemList" />
      </div>
      <div class="flex flex-row space-x-4 p-0.5">
        <NButton v-if="project" @click="goProject">
          {{ project.key }}
        </NButton>

        <NInputGroup style="width: auto">
          <UserSelect
            v-if="allowFilterUsers"
            :user="selectedUserUID"
            :include-system-bot="true"
            :include-all="allowSelectAllUsers"
            @update:user="changeUserUID"
          />
          <NDatePicker
            v-model:value="selectedTimeRange"
            type="datetimerange"
            size="medium"
            :on-confirm="confirmDatePicker"
            :on-clear="clearDatePicker"
            clearable
          >
          </NDatePicker>
          <SearchBox
            :value="state.filterText"
            :placeholder="$t('issue.filter-issue-by-name')"
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
        :page-size="10"
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
        v-if="showClosed"
        session-key="dashboard-closed"
        method="SEARCH"
        :issue-filter="{
          ...issueFilter,
          statusList: [IssueStatus.DONE, IssueStatus.CANCELED],
        }"
        :page-size="10"
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
import AdvancedSearch, { SearchParams } from "@/components/AdvancedSearch.vue";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { UserSelect, SearchBox, TabFilterItem } from "@/components/v2";
import { useCurrentUserV1, useProjectV1Store, useUserStore } from "@/store";
import {
  projectNamePrefix,
  userNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { UNKNOWN_ID, IssueFilter, ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  extractUserUID,
  hasWorkspacePermissionV1,
  projectV1Slug,
  extractProjectResourceName,
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
const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();

const statusList = computed((): string[] =>
  route.query.status ? (route.query.status as string).split(",") : []
);

const autofocus = computed((): boolean => {
  return !!route.query.autofocus;
});

const initSearchParams = computed((): SearchParams => {
  const projectName = project.value?.name ?? "";
  const query = (route.query.query as string) ?? "";

  if (!projectName) {
    return {
      query,
      scopes: [],
    };
  }
  return {
    query,
    scopes: [
      {
        id: "project",
        value: extractProjectResourceName(projectName),
      },
    ],
  };
});

const state = reactive<LocalState>({
  tab: "OPEN",
  filterText: "",
  searchParams: {
    query: "",
    scopes: [],
  },
});

const project = computed(() => {
  if (selectedProjectId.value) {
    return projectV1Store.getProjectByUID(selectedProjectId.value);
  }
  return undefined;
});

const showOpen = computed(
  () => statusList.value.length === 0 || statusList.value.includes("open")
);
const showClosed = computed(
  () => statusList.value.length === 0 || statusList.value.includes("closed")
);

const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  const OPEN: TabFilterItem<TabValue> = {
    value: "OPEN",
    label: t("issue.table.open"),
  };
  const CLOSED: TabFilterItem<TabValue> = {
    value: "CLOSED",
    label: t("issue.table.closed"),
  };
  const list: TabFilterItem<TabValue>[] = [];
  if (showOpen.value) {
    list.push(OPEN);
  }
  if (showClosed.value) {
    list.push(CLOSED);
  }
  return list;
});

const allowFilterUsers = computed(() => {
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }
  return false;
});

const allowSelectAllUsers = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-issue",
    currentUserV1.value.userRole
  );
});

const selectedUserUID = computed((): string => {
  if (!allowFilterUsers.value) {
    // If current user is low-privileged. Don't filter by user id.
    return String(UNKNOWN_ID);
  }

  const id = route.query.user as string;
  if (id) {
    return id;
  }
  return allowSelectAllUsers.value
    ? String(UNKNOWN_ID) // default to 'All' if current user is owner or DBA
    : extractUserUID(currentUserV1.value.name); // default to current user otherwise
});

const selectedTimeRange = computed((): [number, number] => {
  const defaultTimeRange = [
    dayjs().add(-60, "days").toDate().getTime(),
    Date.now(),
  ] as [number, number];
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

const selectedUser = computed(() => {
  const uid = selectedUserUID.value;
  if (uid === String(UNKNOWN_ID)) {
    return;
  }
  return useUserStore().getUserById(uid);
});

const selectedProjectId = computed((): string | undefined => {
  const { project } = route.query;
  return project ? (project as string) : undefined;
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

const changeUserUID = (user: string | undefined) => {
  if (user === String(UNKNOWN_ID)) {
    user = undefined;
  }
  router.replace({
    name: "workspace.issue",
    query: {
      ...route.query,
      user,
    },
  });
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
  state.searchParams = initSearchParams.value;
});

const onSearchParamsUpdate = (params: SearchParams) => {
  state.searchParams = params;
};

const issueFilter = computed((): IssueFilter => {
  const { query, scopes } = state.searchParams;
  const projectScope = scopes.find((s) => s.id === "project");
  const instanceScope = scopes.find((s) => s.id === "instance");
  const typeScope = scopes.find((s) => s.id === "type");

  let instance = "";
  if (instanceScope) {
    instance = `${instanceNamePrefix}${instanceScope.value}`;
  }
  let principal = "";
  if (selectedUser.value) {
    principal = `${userNamePrefix}${selectedUser.value.email}`;
  }
  return {
    query,
    principal,
    instance,
    project: `${projectNamePrefix}${projectScope?.value ?? "-"}`,
    createdTsAfter: selectedTimeRange.value
      ? selectedTimeRange.value[0]
      : undefined,
    createdTsBefore: selectedTimeRange.value
      ? selectedTimeRange.value[1]
      : undefined,
    type: typeScope?.value,
  };
});
</script>
