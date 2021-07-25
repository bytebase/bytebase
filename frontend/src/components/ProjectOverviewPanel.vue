<template>
  <div class="space-y-6">
    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">Databases</p>
      <DatabaseTable :mode="'PROJECT'" :databaseList="databaseList" />
    </div>
    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">Issues</p>
      <IssueTable
        :mode="'PROJECT'"
        :issueSectionList="[
          {
            title: 'In progress',
            list: state.progressIssueList,
          },
          {
            title: 'Recently Closed',
            list: state.closedIssueList,
          },
        ]"
      />
      <router-link
        :to="`/issue?status=closed&project=${project.id}`"
        class="mt-2 flex justify-end normal-link"
      >
        View all closed
      </router-link>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, watchEffect, PropType } from "vue";
import { useStore } from "vuex";
import DatabaseTable from "../components/DatabaseTable.vue";
import IssueTable from "../components/IssueTable.vue";
import { Database, Issue, Project } from "../types";

interface LocalState {
  progressIssueList: Issue[];
  closedIssueList: Issue[];
}

export default {
  name: "ProjectOverviewPanel",
  components: {
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
  setup(props, { emit }) {
    const store = useStore();

    const state = reactive<LocalState>({
      progressIssueList: [],
      closedIssueList: [],
    });

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
      state,
    };
  },
};
</script>
