<template>
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :include-all="true"
        :environment="selectedEnvironment?.uid ?? String(UNKNOWN_ID)"
        @update:environment="changeEnvironmentId($event)"
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
          selectedUserUID && selectedUserUID !== String(UNKNOWN_ID)
            ? selectedUserUID
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
          selectedUserUID && selectedUserUID !== String(UNKNOWN_ID)
            ? selectedUserUID
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
import { NInputGroup, NButton } from "naive-ui";
import { reactive, computed, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import PagedIssueTable from "@/components/Issue/table/PagedIssueTable.vue";
import { EnvironmentTabFilter, UserSelect, SearchBox } from "@/components/v2";
import {
  useCurrentUserV1,
  useEnvironmentV1Store,
  useProjectV1Store,
} from "@/store";
import { Environment } from "@/types/proto/v1/environment_service";
import { IssueTable } from "../components/Issue";
import { type Issue, UNKNOWN_ID } from "../types";
import {
  activeEnvironment,
  extractUserUID,
  hasWorkspacePermissionV1,
  isDatabaseRelatedIssueType,
  projectV1Slug,
} from "../utils";

interface LocalState {
  searchText: string;
}

const router = useRouter();
const route = useRoute();

const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const environmentV1Store = useEnvironmentV1Store();

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

const selectedEnvironment = computed((): Environment | undefined => {
  const { environment } = route.query;
  return environment
    ? environmentV1Store.getEnvironmentByUID(environment as string)
    : undefined;
});

const selectedProjectId = computed((): string | undefined => {
  const { project } = route.query;
  return project ? (project as string) : undefined;
});

const filter = (issue: Issue) => {
  if (selectedEnvironment.value) {
    if (!isDatabaseRelatedIssueType(issue.type)) {
      return false;
    }
    if (
      String(activeEnvironment(issue.pipeline).id) !==
      selectedEnvironment.value.uid
    ) {
      return false;
    }
  }
  const keyword = state.searchText.trim().toLowerCase();
  if (keyword) {
    if (!issue.name.toLowerCase().includes(keyword)) {
      return false;
    }
  }
  return true;
};

const changeEnvironmentId = (environment: string | undefined) => {
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
