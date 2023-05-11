<template>
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :include-all="true"
        :environment="selectedEnvironment?.id ?? UNKNOWN_ID"
        @update:environment="changeEnvironmentId($event)"
      />
      <div class="flex flex-row space-x-4">
        <NButton v-if="project" @click="goProject">
          {{ project.key }}
        </NButton>

        <NInputGroup style="width: auto">
          <PrincipalSelect
            v-if="allowFilterUsers"
            :principal="selectedPrincipalId"
            :include-system-bot="true"
            :include-all="allowSelectAllUsers"
            @update:principal="changePrincipalId"
          />
          <SearchBox
            :value="state.searchText"
            :placeholder="$t('issue.search-issue-name')"
            :autofocus="true"
            @update:value="changeSearchText($event)"
          />
        </NInputGroup>
      </div>
    </div>

    <!-- show all OPEN issues with pageSize=10  -->
    <PagedIssueTable
      v-if="showOpen"
      session-key="dashboard-open"
      :issue-find="{
        statusList: ['OPEN'],
        principalId:
          selectedPrincipalId && selectedPrincipalId !== UNKNOWN_ID
            ? selectedPrincipalId
            : undefined,
        projectId: selectedProjectId,
      }"
      :page-size="10"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          :left-bordered="false"
          :right-bordered="false"
          :top-bordered="true"
          :bottom-bordered="true"
          :show-placeholder="!loading"
          :title="$t('issue.table.open')"
          :issue-list="issueList.filter(filter)"
        />
      </template>
    </PagedIssueTable>

    <!-- show all DONE and CANCELED issues with pageSize=10 -->
    <PagedIssueTable
      v-if="showClosed"
      session-key="dashboard-closed"
      :issue-find="{
        statusList: ['DONE', 'CANCELED'],
        principalId:
          selectedPrincipalId && selectedPrincipalId !== UNKNOWN_ID
            ? selectedPrincipalId
            : undefined,
        projectId: selectedProjectId,
      }"
      :page-size="10"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          class="-mt-px"
          :left-bordered="false"
          :right-bordered="false"
          :top-bordered="true"
          :bottom-bordered="true"
          :show-placeholder="!loading"
          :title="$t('issue.table.closed')"
          :issue-list="issueList.filter(filter)"
        />
      </template>
    </PagedIssueTable>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NInputGroup, NButton } from "naive-ui";

import {
  EnvironmentTabFilter,
  PrincipalSelect,
  SearchBox,
} from "@/components/v2";
import { IssueTable } from "../components/Issue";
import {
  type Environment,
  type EnvironmentId,
  type Issue,
  type PrincipalId,
  type ProjectId,
  UNKNOWN_ID,
} from "../types";
import {
  activeEnvironment,
  hasWorkspacePermission,
  isDatabaseRelatedIssueType,
  projectV1Slug,
} from "../utils";
import {
  useCurrentUser,
  useEnvironmentStore,
  useProjectV1Store,
} from "@/store";
import PagedIssueTable from "@/components/Issue/table/PagedIssueTable.vue";

interface LocalState {
  searchText: string;
}

const router = useRouter();
const route = useRoute();

const currentUser = useCurrentUser();
const projectV1Store = useProjectV1Store();
const environmentStore = useEnvironmentStore();

const statusList = computed((): string[] =>
  route.query.status ? (route.query.status as string).split(",") : []
);

const state = reactive<LocalState>({
  searchText: "",
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
    hasWorkspacePermission(
      "bb.permission.workspace.manage-issue",
      currentUser.value.role
    )
  ) {
    return true;
  }
  return false;
});

const allowSelectAllUsers = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-issue",
    currentUser.value.role
  );
});

const selectedPrincipalId = computed((): PrincipalId => {
  if (!allowFilterUsers.value) {
    // If current user is low-privileged. Don't filter by user id.
    return UNKNOWN_ID;
  }

  const id = parseInt(route.query.user as string, 10);
  if (id >= 0) {
    return id;
  }
  return allowSelectAllUsers.value
    ? UNKNOWN_ID // default to 'All' if current user is owner or DBA
    : currentUser.value.id; // default to current user otherwise
});

const selectedEnvironment = computed((): Environment | undefined => {
  const { environment } = route.query;
  return environment
    ? environmentStore.getEnvironmentById(parseInt(environment as string, 10))
    : undefined;
});

const selectedProjectId = computed((): ProjectId | undefined => {
  const { project } = route.query;
  return project ? parseInt(project as string, 10) : undefined;
});

const filter = (issue: Issue) => {
  if (selectedEnvironment.value) {
    if (!isDatabaseRelatedIssueType(issue.type)) {
      return false;
    }
    if (activeEnvironment(issue.pipeline).id !== selectedEnvironment.value.id) {
      return false;
    }
  }
  const keyword = state.searchText.trim();
  if (keyword) {
    if (!issue.name.toLowerCase().includes(keyword)) {
      return false;
    }
  }
  return true;
};

const changeEnvironmentId = (environment: EnvironmentId | undefined) => {
  if (environment && environment !== UNKNOWN_ID) {
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

const changePrincipalId = (user: PrincipalId | undefined) => {
  if (user === UNKNOWN_ID) {
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

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
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
</script>
