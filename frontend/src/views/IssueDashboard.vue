<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <!-- eslint-disable vue/attribute-hyphenation -->
      <EnvironmentTabFilter
        :selectedId="selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <div class="flex flex-row space-x-4">
        <button
          v-if="project"
          class="px-4 cursor-pointer rounded-md text-control text-sm bg-link-hover focus:outline-none hover:underline"
          @click.prevent="goProject"
        >
          {{ project.key }}
        </button>
        <!-- eslint-disable vue/attribute-hyphenation -->
        <MemberSelect
          class="w-72"
          :show-all="true"
          :show-system-bot="true"
          :selected-id="selectedPrincipalId"
          @select-principal-id="selectPrincipal"
        />
        <BBTableSearch
          ref="searchField"
          :placeholder="$t('issue.search-issue-name')"
          @change-text="(text: string) => changeSearchText(text)"
        />
      </div>
    </div>

    <!-- show all OPEN issues with pageSize=10  -->
    <PagedIssueTable
      v-if="showOpen"
      session-key="dashboard-open"
      :issue-find="{
        statusList: ['OPEN'],
        principalId: selectedPrincipalId > 0 ? selectedPrincipalId : undefined,
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
        principalId: selectedPrincipalId > 0 ? selectedPrincipalId : undefined,
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
import { useRoute, useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import { IssueTable } from "../components/Issue";
import MemberSelect from "../components/MemberSelect.vue";
import { EMPTY_ID, Environment, Issue, PrincipalId, ProjectId } from "../types";
import { reactive, ref, computed, onMounted } from "vue";
import {
  activeEnvironment,
  hasWorkspacePermission,
  projectSlug,
} from "../utils";
import { useCurrentUser, useEnvironmentStore, useProjectStore } from "@/store";
import PagedIssueTable from "@/components/Issue/PagedIssueTable.vue";

interface LocalState {
  searchText: string;
}

const searchField = ref();

const router = useRouter();
const route = useRoute();

const currentUser = useCurrentUser();
const projectStore = useProjectStore();
const environmentStore = useEnvironmentStore();

const statusList = computed((): string[] =>
  route.query.status ? (route.query.status as string).split(",") : []
);

const state = reactive<LocalState>({
  searchText: "",
});

const showOpen = computed(
  () => statusList.value.length === 0 || statusList.value.includes("open")
);
const showClosed = computed(
  () => statusList.value.length === 0 || statusList.value.includes("closed")
);

const selectedPrincipalId = computed((): PrincipalId => {
  const id = parseInt(route.query.user as string, 10);
  if (id >= 0) {
    return id;
  }
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-issue",
    currentUser.value.role
  )
    ? EMPTY_ID // default to 'All' if current user is owner or DBA
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

onMounted(() => {
  // Focus on the internal search field when mounted
  searchField.value.$el.querySelector("#search").focus();
});

const project = computed(() => {
  if (selectedProjectId.value) {
    return projectStore.getProjectById(selectedProjectId.value);
  }
  return undefined;
});

const filter = (issue: Issue) => {
  if (selectedEnvironment.value) {
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

const selectEnvironment = (environment: Environment) => {
  if (environment) {
    router.replace({
      name: "workspace.issue",
      query: {
        ...route.query,
        environment: environment.id,
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

const selectPrincipal = (principalId: PrincipalId) => {
  router.replace({
    name: "workspace.issue",
    query: {
      ...route.query,
      user: principalId,
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
      projectSlug: projectSlug(project.value),
    },
  });
};
</script>
