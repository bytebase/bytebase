<template>
  <div class="space-y-6">
    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("project.overview.recent-activity") }}
      </p>
      <ActivityTable :activity-list="state.activityList" />
      <router-link
        :to="`#activity`"
        class="mt-2 flex justify-end normal-link"
        exact-active-class=""
      >
        {{ $t("project.overview.view-all") }}
      </router-link>
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
      <BBAttention
        v-if="project.id == DEFAULT_PROJECT_ID"
        :style="`INFO`"
        :title="$t('project.overview.info-slot-content')"
      />

      <YAxisRadioGroup
        v-if="isTenantProject && state.yAxisLabel"
        v-model:label="state.yAxisLabel"
        :label-list="selectableLabelList"
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
            :label-list="labelList"
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
      <IssueTable
        :mode="'PROJECT'"
        :issue-section-list="[
          {
            title: $t('project.overview.in-progress'),
            list: state.progressIssueList,
          },
          {
            title: $t('project.overview.recently-closed'),
            list: state.closedIssueList,
          },
        ]"
      />
      <router-link
        :to="`/issue?status=closed&project=${project.id}`"
        class="mt-2 flex justify-end normal-link"
      >
        {{ $t("project.overview.view-all-closed") }}
      </router-link>
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
} from "vue";
import ActivityTable from "../components/ActivityTable.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import TenantDatabaseTable, { YAxisRadioGroup } from "./TenantDatabaseTable";
import { IssueTable } from "../components/Issue";
import {
  Activity,
  Database,
  Issue,
  Project,
  DEFAULT_PROJECT_ID,
  LabelKeyType,
} from "../types";
import { findDefaultGroupByLabel } from "../utils";
import { NSpin } from "naive-ui";
import { useActivityStore, useIssueStore, useLabelStore } from "@/store";
import { storeToRefs } from "pinia";

// Show at most 5 activity
const ACTIVITY_LIMIT = 5;

interface LocalState {
  activityList: Activity[];
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
    const labelStore = useLabelStore();

    const state = reactive<LocalState>({
      activityList: [],
      progressIssueList: [],
      closedIssueList: [],
      databaseNameFilter: "",
      xAxisLabel: "bb.environment",
      yAxisLabel: undefined,
    });

    const prepareActivityList = () => {
      useActivityStore()
        .fetchActivityListForProject({
          projectId: props.project.id,
          limit: ACTIVITY_LIMIT,
        })
        .then((list) => {
          state.activityList = list;
        });
    };

    const prepareIssueList = () => {
      useIssueStore()
        .fetchIssueList({
          projectId: props.project.id,
        })
        .then((issueList: Issue[]) => {
          state.progressIssueList = [];
          state.closedIssueList = [];
          for (const issue of issueList) {
            // "OPEN"
            if (issue.status === "OPEN") {
              state.progressIssueList.push(issue);
            }
            // "DONE" or "CANCELED"
            else if (issue.status === "DONE" || issue.status === "CANCELED") {
              state.closedIssueList.push(issue);
            }
          }
        });
    };

    const isTenantProject = computed((): boolean => {
      return props.project.tenantMode === "TENANT";
    });

    const prepareLabelList = () => {
      if (!isTenantProject.value) return;
      labelStore.fetchLabelList();
    };

    const prepare = () => {
      prepareActivityList();
      prepareIssueList();
      prepareLabelList();
    };

    watchEffect(prepare);

    const { labelList } = storeToRefs(labelStore);

    const filteredDatabaseList = computed(() => {
      const filter = state.databaseNameFilter.toLocaleLowerCase();
      if (!filter) return props.databaseList;

      return props.databaseList.filter((database) =>
        database.name.toLowerCase().includes(filter)
      );
    });

    // make "bb.environment" non-selectable because it was already specified to the x-axis
    const selectableLabelList = computed(() => {
      const excludes = new Set([state.xAxisLabel]);
      return labelList.value.filter((label) => !excludes.has(label.key));
    });

    watchEffect(() => {
      state.yAxisLabel = findDefaultGroupByLabel(
        selectableLabelList.value,
        filteredDatabaseList.value
      );
    });

    return {
      DEFAULT_PROJECT_ID,
      state,
      isTenantProject,
      filteredDatabaseList,
      labelList,
      selectableLabelList,
    };
  },
});
</script>
