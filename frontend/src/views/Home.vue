<template>
  <div class="flex flex-col">
    <div class="px-5 py-2 flex justify-between items-center">
      <!-- eslint-disable vue/attribute-hyphenation -->
      <EnvironmentTabFilter
        :selectedId="selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('issue.search-issue-name')"
        @change-text="(text: string) => changeSearchText(text)"
      />
    </div>

    <!-- show OPEN Assigned issues with pageSize=10 -->
    <PagedIssueTable
      session-key="home-assigned"
      :issue-find="{
        statusList: ['OPEN'],
        assigneeId: currentUser.id,
      }"
      :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          :left-bordered="false"
          :right-bordered="false"
          :show-placeholder="!loading"
          :title="$t('common.assigned')"
          :issue-list="issueList.filter(keywordAndEnvironmentFilter)"
        />
      </template>
    </PagedIssueTable>

    <!-- show OPEN Created issues with pageSize=10 -->
    <PagedIssueTable
      session-key="home-created"
      :issue-find="{
        statusList: ['OPEN'],
        creatorId: currentUser.id,
      }"
      :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          class="-mt-px"
          :left-bordered="false"
          :right-bordered="false"
          :show-placeholder="!loading"
          :title="$t('common.created')"
          :issue-list="issueList.filter(keywordAndEnvironmentFilter)"
        />
      </template>
    </PagedIssueTable>

    <!-- show OPEN Subscribed issues with pageSize=10 -->
    <PagedIssueTable
      session-key="home-subscribed"
      :issue-find="{
        statusList: ['OPEN'],
        subscriberId: currentUser.id,
      }"
      :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          class="-mt-px"
          :left-bordered="false"
          :right-bordered="false"
          :show-placeholder="!loading"
          :title="$t('common.subscribed')"
          :issue-list="issueList.filter(keywordAndEnvironmentFilter)"
        />
      </template>
    </PagedIssueTable>

    <!-- show the first 5 DONE or CANCELED issues -->
    <!-- But won't show "Load more", since we have a "View all closed" link below -->
    <PagedIssueTable
      session-key="home-closed"
      :issue-find="{
        statusList: ['DONE', 'CANCELED'],
        principalId: currentUser.id,
      }"
      :page-size="MAX_CLOSED_ISSUE"
      :hide-load-more="true"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          class="-mt-px"
          :left-bordered="false"
          :right-bordered="false"
          :show-placeholder="!loading"
          :title="$t('project.overview.recently-closed')"
          :issue-list="issueList.filter(keywordAndEnvironmentFilter)"
        />
      </template>
    </PagedIssueTable>
  </div>
  <div class="w-full flex justify-end mt-2 px-4">
    <router-link to="/issue?status=closed" class="normal-link">
      {{ $t("project.overview.view-all-closed") }}
    </router-link>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import { IssueTable } from "../components/Issue";
import { activeEnvironment } from "../utils";
import { Environment, Issue } from "../types";
import { useCurrentUser, useEnvironmentStore } from "@/store";
import PagedIssueTable from "@/components/Issue/PagedIssueTable.vue";

interface LocalState {
  searchText: string;
}

const OPEN_ISSUE_LIST_PAGE_SIZE = 10;
const MAX_CLOSED_ISSUE = 5;

const searchField = ref();

const environmentStore = useEnvironmentStore();
const router = useRouter();
const route = useRoute();

const state = reactive<LocalState>({
  searchText: "",
});

const currentUser = useCurrentUser();

const selectedEnvironment = computed(() => {
  const { environment } = route.query;
  return environment
    ? environmentStore.getEnvironmentById(parseInt(environment as string, 10))
    : undefined;
});

const keywordAndEnvironmentFilter = (issue: Issue) => {
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
      name: "workspace.home",
      query: {
        ...route.query,
        environment: environment.id,
      },
    });
  } else {
    router.replace({
      name: "workspace.home",
      query: {
        ...route.query,
        environment: undefined,
      },
    });
  }
};

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};

onMounted(() => {
  // Focus on the internal search field when mounted
  searchField.value.$el.querySelector("#search").focus();
});
</script>
