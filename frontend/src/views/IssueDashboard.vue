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

    <div class="px-4 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :include-all="true"
        :environment="selectedEnvironment?.name"
        @update:environment="changeEnvironment($event)"
      />
      <div class="flex flex-row space-x-4">
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
          <SearchBox
            :value="state.filterText"
            :placeholder="$t('issue.filter-issue-by-name')"
            :autofocus="false"
            @update:value="state.filterText = $event"
          />
        </NInputGroup>
      </div>
    </div>

    <!-- show all OPEN issues with pageSize=10  -->
    <PagedIssueTableV1
      v-if="showOpen"
      session-key="dashboard-open"
      :issue-filter="{
        ...issueFilter,
        statusList: [IssueStatus.OPEN],
      }"
      :page-size="10"
    >
      <template #table="{ issueList, loading }">
        <IssueTableV1
          :left-bordered="false"
          :right-bordered="false"
          :top-bordered="true"
          :bottom-bordered="true"
          :show-placeholder="!loading"
          :title="$t('issue.table.open')"
          :issue-list="issueList.filter(filter)"
          :highlight-text="state.searchParams.query"
        />
      </template>
    </PagedIssueTableV1>

    <!-- show all DONE and CANCELED issues with pageSize=10 -->
    <PagedIssueTableV1
      v-if="showClosed"
      session-key="dashboard-closed"
      :issue-filter="{
        ...issueFilter,
        statusList: [IssueStatus.DONE, IssueStatus.CANCELED],
      }"
      :page-size="10"
    >
      <template #table="{ issueList, loading }">
        <IssueTableV1
          class="-mt-px"
          :left-bordered="false"
          :right-bordered="false"
          :top-bordered="true"
          :bottom-bordered="true"
          :show-placeholder="!loading"
          :title="$t('issue.table.closed')"
          :issue-list="issueList.filter(filter)"
          :highlight-text="state.searchParams.query"
        />
      </template>
    </PagedIssueTableV1>
  </div>
</template>

<script lang="ts" setup>
import { NInputGroup, NButton } from "naive-ui";
import { reactive, computed, watchEffect, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { SearchParams } from "@/components/AdvancedSearch.vue";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { EnvironmentTabFilter, UserSelect, SearchBox } from "@/components/v2";
import {
  useCurrentUserV1,
  useEnvironmentV1Store,
  useProjectV1Store,
  useUserStore,
} from "@/store";
import { projectNamePrefix, userNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_ID, IssueFilter, ComposedIssue } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  extractUserUID,
  hasWorkspacePermissionV1,
  isDatabaseRelatedIssue,
  activeEnvironmentInRollout,
  projectV1Slug,
} from "@/utils";

interface LocalState {
  filterText: string;
  searchParams: SearchParams;
}

const router = useRouter();
const route = useRoute();

const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const environmentV1Store = useEnvironmentV1Store();

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
        value: projectName,
      },
    ],
  };
});

const state = reactive<LocalState>({
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

const selectedUser = computed(() => {
  const uid = selectedUserUID.value;
  if (uid === String(UNKNOWN_ID)) {
    return;
  }
  return useUserStore().getUserById(uid);
});

const selectedEnvironment = computed((): Environment | undefined => {
  const { environment } = route.query;
  return environment
    ? environmentV1Store.getEnvironmentByName(environment as string)
    : undefined;
});

const selectedProjectId = computed((): string | undefined => {
  const { project } = route.query;
  return project ? (project as string) : undefined;
});

const filter = (issue: ComposedIssue) => {
  if (selectedEnvironment.value) {
    if (!isDatabaseRelatedIssue(issue)) {
      return false;
    }
    if (
      activeEnvironmentInRollout(issue.rolloutEntity) !==
      selectedEnvironment.value.name
    ) {
      return false;
    }
  }
  const keyword = state.filterText.trim().toLowerCase();
  if (keyword) {
    if (!issue.title.toLowerCase().includes(keyword)) {
      return false;
    }
  }
  return true;
};

const changeEnvironment = (environment: string | undefined) => {
  if (environment && environment !== String(UNKNOWN_ID)) {
    router.replace({
      name: "workspace.issue",
      query: {
        ...route.query,
        environment,
      },
    });
  } else {
    router.replace({
      name: "workspace.issue",
      query: {
        ...route.query,
        environment: undefined,
      },
    });
  }
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
  const project = projectScope?.value ?? `${projectNamePrefix}-`;
  let principal = "";
  if (selectedUser.value) {
    principal = `${userNamePrefix}${selectedUser.value.email}`;
  }
  return {
    project,
    query,
    principal,
  };
});
</script>
