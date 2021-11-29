<template>
  <div class="flex flex-col">
    <div class="px-5 py-2 flex justify-between items-center">
      <!-- eslint-disable vue/attribute-hyphenation -->
      <EnvironmentTabFilter
        :selectedID="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="'Search issue name'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <IssueTable
      :left-bordered="false"
      :right-bordered="false"
      :top-bordered="true"
      :bottom-bordered="true"
      :issue-section-list="[
        {
          title: 'Assigned',
          list: filteredList(state.assignedList).sort(openIssueSorter),
        },
        {
          title: 'Created',
          list: filteredList(state.createdList).sort(openIssueSorter),
        },
        {
          title: 'Subscribed',
          list: filteredList(state.subscribeList).sort(openIssueSorter),
        },
        {
          title: 'Recently Closed',
          list: filteredList(state.closedList).sort((a, b) => {
            return b.updatedTs - a.updatedTs;
          }),
        },
      ]"
    />
  </div>
  <router-link
    to="/issue?status=closed"
    class="mt-2 px-4 flex justify-end normal-link"
  >
    View all closed
  </router-link>
</template>

<script lang="ts">
import { watchEffect, computed, onMounted, reactive, ref } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import IssueTable from "../components/IssueTable.vue";
import { activeEnvironment, activeTask } from "../utils";
import { Environment, Issue, TaskStatus, UNKNOWN_ID } from "../types";

// Show at most 10 recently closed issues
const MAX_CLOSED_ISSUE_COUNT = 10;

interface LocalState {
  createdList: Issue[];
  assignedList: Issue[];
  subscribeList: Issue[];
  closedList: Issue[];
  searchText: string;
  selectedEnvironment?: Environment;
}

export default {
  name: "Home",
  components: {
    EnvironmentTabFilter,
    IssueTable,
  },
  setup() {
    const searchField = ref();

    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      createdList: [],
      assignedList: [],
      subscribeList: [],
      closedList: [],
      searchText: "",
      selectedEnvironment: router.currentRoute.value.query.environment
        ? store.getters["environment/environmentByID"](
            router.currentRoute.value.query.environment
          )
        : undefined,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    const prepareIssueList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store
          .dispatch("issue/fetchIssueList", {
            issueStatusList: ["OPEN"],
            userID: currentUser.value.id,
          })
          .then((issueList: Issue[]) => {
            state.assignedList = [];
            state.createdList = [];
            state.subscribeList = [];
            for (const issue of issueList) {
              if (issue.assignee?.id === currentUser.value.id) {
                state.assignedList.push(issue);
              } else if (issue.creator.id === currentUser.value.id) {
                state.createdList.push(issue);
              } else if (
                issue.subscriberIDList.includes(currentUser.value.id)
              ) {
                state.subscribeList.push(issue);
              }
            }
          });

        store
          .dispatch("issue/fetchIssueList", {
            issueStatusList: ["DONE", "CANCELED"],
            userID: currentUser.value.id,
            limit: MAX_CLOSED_ISSUE_COUNT,
          })
          .then((issueList: Issue[]) => {
            state.closedList = [];
            for (const issue of issueList) {
              if (
                issue.creator.id === currentUser.value.id ||
                issue.assignee?.id === currentUser.value.id ||
                issue.subscriberIDList.includes(currentUser.value.id)
              ) {
                state.closedList.push(issue);
              }
            }
          });
      }
    };

    watchEffect(prepareIssueList);

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
      if (environment) {
        router.replace({
          name: "workspace.home",
          query: { environment: environment.id },
        });
      } else {
        router.replace({ name: "workspace.home" });
      }
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const filteredList = (list: Issue[]) => {
      if (!state.selectedEnvironment && !state.searchText) {
        // Select "All"
        return list;
      }
      return list.filter((issue) => {
        return (
          (!state.selectedEnvironment ||
            activeEnvironment(issue.pipeline).id ===
              state.selectedEnvironment.id) &&
          (!state.searchText ||
            issue.name.toLowerCase().includes(state.searchText.toLowerCase()))
        );
      });
    };

    const openIssueSorter = (a: Issue, b: Issue) => {
      const statusOrder = (status: TaskStatus) => {
        switch (status) {
          case "FAILED":
            return 0;
          case "PENDING_APPROVAL":
            return 1;
          case "PENDING":
            return 2;
          case "RUNNING":
            return 3;
          case "DONE":
            return 4;
          case "CANCELED":
            return 5;
        }
      };
      const aStatusOrder = statusOrder(activeTask(a.pipeline).status);
      const bStatusOrder = statusOrder(activeTask(b.pipeline).status);
      if (aStatusOrder == bStatusOrder) {
        return b.updatedTs - a.updatedTs;
      }
      return aStatusOrder - bStatusOrder;
    };

    return {
      searchField,
      state,
      filteredList,
      selectEnvironment,
      changeSearchText,
      openIssueSorter,
    };
  },
};
</script>
