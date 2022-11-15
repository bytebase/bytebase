<template>
  <div class="space-y-6">
    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("project.overview.recent-activity") }}
      </p>
      <div class="relative">
        <ActivityTable :activity-list="state.activityList" />
        <div
          v-if="state.isFetchingActivityList"
          class="absolute inset-0 flex flex-col items-center justify-center bg-white/70"
        >
          <BBSpin />
        </div>
        <div class="w-full flex justify-end mt-2 px-4">
          <router-link
            :to="`#activity`"
            class="normal-link"
            exact-active-class=""
          >
            {{ $t("project.overview.view-all") }}
          </router-link>
        </div>
      </div>
    </div>

    <div class="space-y-2">
      <div
        class="text-lg font-medium leading-7 text-main flex items-center justify-between"
      >
        {{ $t("common.database") }}
        <div v-if="isTenantProject">
          <label for="search" class="sr-only">Search</label>
          <div class="relative rounded-md shadow-sm">
            <div
              class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"
              aria-hidden="true"
            >
              <heroicons-solid:search class="mr-3 h-4 w-4 text-control" />
            </div>
            <input
              v-model="state.databaseNameFilter"
              type="text"
              autocomplete="off"
              name="search"
              class="focus:ring-main focus:border-main block w-full pl-9 sm:text-sm border-control-border rounded-md"
              :placeholder="$t('database.search-database-name')"
            />
          </div>
        </div>
      </div>

      <YAxisRadioGroup
        v-if="isTenantProject && state.yAxisLabel"
        v-model:label="state.yAxisLabel"
        :excluded-key-list="excludedKeyList"
        class="text-sm font-normal py-1"
      />

      <template v-if="databaseList.length > 0">
        <template v-if="isTenantProject">
          <TenantDatabaseTable
            v-if="state.yAxisLabel"
            :database-list="filteredDatabaseList"
            :project="project"
            :x-axis-label="state.xAxisLabel"
            :y-axis-label="state.yAxisLabel"
          />
          <div v-else class="w-full h-40 flex justify-center items-center">
            <NSpin />
          </div>
        </template>
        <DatabaseTable v-else :mode="'PROJECT'" :database-list="databaseList" />
      </template>
      <div v-else class="text-center textinfolabel">
        <i18n-t keypath="project.overview.no-db-prompt" tag="p">
          <template #newDb>
            <span class="text-main">{{
              $t("quick-action.new-db")
            }}</span></template
          >
          <template #transferInDb>
            <span class="text-main">{{
              $t("quick-action.transfer-in-db")
            }}</span></template
          >
        </i18n-t>
      </div>
    </div>

    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("common.issue") }}
      </p>

      <!-- show OPEN issues with pageSize=10 -->
      <div>
        <PagedIssueTable
          session-key="project-open"
          :issue-find="{
            statusList: ['OPEN'],
            projectId: project.id,
          }"
          :page-size="10"
        >
          <template #table="{ issueList, loading }">
            <IssueTable
              :mode="'PROJECT'"
              :title="$t('project.overview.in-progress')"
              :issue-list="issueList"
              :show-placeholder="!loading"
            />
          </template>
        </PagedIssueTable>

        <!-- show the first 5 DONE or CANCELED issues -->
        <!-- But won't show "Load more", since we have a "View all closed" link below -->
        <PagedIssueTable
          session-key="project-closed"
          :issue-find="{
            statusList: ['DONE', 'CANCELED'],
            projectId: project.id,
          }"
          :page-size="5"
          :hide-load-more="true"
        >
          <template #table="{ issueList, loading }">
            <IssueTable
              class="-mt-px"
              :mode="'PROJECT'"
              :title="$t('project.overview.recently-closed')"
              :issue-list="issueList"
              :show-placeholder="!loading"
            />
          </template>
        </PagedIssueTable>

        <div class="w-full flex justify-end mt-2 px-4">
          <router-link
            :to="`/issue?status=closed&project=${project.id}`"
            class="normal-link"
          >
            {{ $t("project.overview.view-all-closed") }}
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import {
  reactive,
  watchEffect,
  PropType,
  computed,
  defineComponent,
  watch,
} from "vue";
import { NSpin } from "naive-ui";
import ActivityTable from "../components/ActivityTable.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import TenantDatabaseTable, { YAxisRadioGroup } from "./TenantDatabaseTable";
import { IssueTable } from "../components/Issue";
import { Activity, Database, Issue, Project, LabelKeyType } from "../types";
import { findDefaultGroupByLabel } from "../utils";
import { useActivityStore } from "@/store";
import PagedIssueTable from "@/components/Issue/PagedIssueTable.vue";

// Show at most 5 activity
const ACTIVITY_LIMIT = 5;

interface LocalState {
  activityList: Activity[];
  isFetchingActivityList: boolean;
  progressIssueList: Issue[];
  closedIssueList: Issue[];
  databaseNameFilter: string;
  xAxisLabel: LabelKeyType;
  yAxisLabel: LabelKeyType | undefined;
}

export default defineComponent({
  name: "ProjectOverviewPanel",
  components: {
    ActivityTable,
    DatabaseTable,
    TenantDatabaseTable,
    IssueTable,
    YAxisRadioGroup,
    NSpin,
    PagedIssueTable,
  },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
    databaseList: {
      required: true,
      type: Object as PropType<Database[]>,
    },
  },
  setup(props) {
    const state = reactive<LocalState>({
      activityList: [],
      isFetchingActivityList: false,
      progressIssueList: [],
      closedIssueList: [],
      databaseNameFilter: "",
      xAxisLabel: "bb.environment",
      yAxisLabel: undefined,
    });
    const activityStore = useActivityStore();

    const prepareActivityList = () => {
      state.isFetchingActivityList = true;
      state.activityList = [];
      const requests = [
        activityStore.fetchActivityListForDatabaseByProjectId({
          projectId: props.project.id,
          limit: ACTIVITY_LIMIT,
        }),
        activityStore.fetchActivityListForProject({
          projectId: props.project.id,
          limit: ACTIVITY_LIMIT,
        }),
      ];

      Promise.all(requests).then((lists) => {
        const flattenList = lists.flatMap((list) => list);
        flattenList.sort((a, b) => -(a.createdTs - b.createdTs)); // by createdTs DESC
        state.activityList = flattenList.slice(0, ACTIVITY_LIMIT);

        state.isFetchingActivityList = false;
      });
    };

    const isTenantProject = computed((): boolean => {
      return props.project.tenantMode === "TENANT";
    });

    const prepare = () => {
      prepareActivityList();
    };

    watch(() => props.project.id, prepare, { immediate: true });

    const filteredDatabaseList = computed(() => {
      const filter = state.databaseNameFilter.toLocaleLowerCase();
      if (!filter) return props.databaseList;

      return props.databaseList.filter((database) =>
        database.name.toLowerCase().includes(filter)
      );
    });

    const excludedKeyList = computed(() => [state.xAxisLabel]);

    watchEffect(() => {
      state.yAxisLabel = findDefaultGroupByLabel(
        filteredDatabaseList.value,
        excludedKeyList.value
      );
    });

    return {
      state,
      isTenantProject,
      filteredDatabaseList,
      excludedKeyList,
    };
  },
});
</script>
