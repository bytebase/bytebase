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
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("common.database") }}
      </p>
      <BBAttention
        v-if="project.id == DEFAULT_PROJECT_ID"
        :style="`INFO`"
        :title="$t('project.overview.info-slot-content')"
      />
      <DatabaseTable
        v-if="databaseList.length > 0"
        :mode="'PROJECT'"
        :database-list="databaseList"
      />
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
import { reactive, watchEffect, PropType } from "vue";
import { useStore } from "vuex";
import ActivityTable from "../components/ActivityTable.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import { IssueTable } from "../components/Issue";
import {
  Activity,
  Database,
  Issue,
  Project,
  DEFAULT_PROJECT_ID,
} from "../types";

// Show at most 5 activity
const ACTIVITY_LIMIT = 5;

interface LocalState {
  activityList: Activity[];
  progressIssueList: Issue[];
  closedIssueList: Issue[];
}

export default {
  name: "ProjectOverviewPanel",
  components: {
    ActivityTable,
    DatabaseTable,
    IssueTable,
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
    const store = useStore();

    const state = reactive<LocalState>({
      activityList: [],
      progressIssueList: [],
      closedIssueList: [],
    });

    const prepareActivityList = () => {
      store
        .dispatch("activity/fetchActivityListForProject", {
          projectId: props.project.id,
          limit: ACTIVITY_LIMIT,
        })
        .then((list) => {
          state.activityList = list;
        });
    };

    watchEffect(prepareActivityList);

    const prepareIssueList = () => {
      store
        .dispatch("issue/fetchIssueList", {
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

    watchEffect(prepareIssueList);

    return {
      DEFAULT_PROJECT_ID,
      state,
    };
  },
};
</script>
