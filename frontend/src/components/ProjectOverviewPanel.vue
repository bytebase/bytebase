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
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive, watchEffect, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DatabaseTable from "../components/DatabaseTable.vue";
import IssueTable from "../components/IssueTable.vue";
import { Issue, Project } from "../types";
import { sortDatabaseList } from "../utils";

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
  },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      progressIssueList: [],
      closedIssueList: [],
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"](["NORMAL"]);
    });

    const databaseList = computed(() => {
      const list = store.getters["database/databaseListByProjectId"](
        props.project.id
      );
      return sortDatabaseList(list, environmentList.value);
    });

    const prepareDatabaseList = () => {
      store.dispatch("database/fetchDatabaseListByProjectId", props.project.id);
    };

    watchEffect(prepareDatabaseList);

    const prepareIssueList = () => {
      store
        .dispatch("issue/fetchIssueListForProject", props.project.id)
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
      databaseList,
    };
  },
};
</script>
