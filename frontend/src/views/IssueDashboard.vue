<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :selectedId="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="'Search issue name'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <IssueTable
      :leftBordered="false"
      :rightBordered="false"
      :topBordered="true"
      :bottomBordered="true"
      :issueSectionList="sectionList"
    />
  </div>
</template>

<script lang="ts">
import { reactive, ref } from "@vue/reactivity";
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import IssueTable from "../components/IssueTable.vue";
import { Environment, Issue, UNKNOWN_ID } from "../types";
import { computed, onMounted, watchEffect } from "@vue/runtime-core";
import { activeEnvironment } from "../utils";
import { BBTableSectionDataSource } from "../bbkit/types";

interface LocalState {
  showOpen: boolean;
  showClosed: boolean;
  openList: Issue[];
  closedList: Issue[];
  searchText: string;
  selectedEnvironment?: Environment;
}

export default {
  name: "IssueDashboard",
  components: { EnvironmentTabFilter, IssueTable },
  setup(props, ctx) {
    const searchField = ref();

    const store = useStore();
    const router = useRouter();

    const statusList: string[] = router.currentRoute.value.query.status
      ? (router.currentRoute.value.query.status as string).split(",")
      : [];

    const state = reactive<LocalState>({
      showOpen: statusList.length == 0 || statusList.includes("open"),
      showClosed: statusList.length == 0 || statusList.includes("closed"),
      openList: [],
      closedList: [],
      searchText: "",
      selectedEnvironment: router.currentRoute.value.query.environment
        ? store.getters["environment/environmentById"](
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
        // We call open and close separately because normally the number of open issues is limited
        // while the closed issues could be a lot.
        if (state.showOpen) {
          store
            .dispatch("issue/fetchIssueListForUser", {
              userId: currentUser.value.id,
              issueStatusList: ["OPEN"],
            })
            .then((issueList: Issue[]) => {
              state.openList = issueList;
            });
        }

        if (state.showClosed) {
          store
            .dispatch("issue/fetchIssueListForUser", {
              userId: currentUser.value.id,
              issueStatusList: ["DONE", "CANCELED"],
            })
            .then((issueList: Issue[]) => {
              state.closedList = [];
              for (const issue of issueList) {
                // "DONE" or "CANCELED"
                if (issue.status === "DONE" || issue.status === "CANCELED") {
                  if (
                    issue.creator.id === currentUser.value.id ||
                    issue.assignee?.id === currentUser.value.id ||
                    issue.subscriberIdList.includes(currentUser.value.id)
                  ) {
                    state.closedList.push(issue);
                  }
                }
              }
            });
        }
      }
    };

    watchEffect(prepareIssueList);

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
      if (environment) {
        router.replace({
          name: "workspace.issue",
          query: { environment: environment.id },
        });
      } else {
        router.replace({ name: "workspace.issue" });
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

    const sectionList = computed((): BBTableSectionDataSource<Issue>[] => {
      const list = [];
      if (state.showOpen) {
        list.push({
          title: "Open",
          list: filteredList(state.openList).sort((a, b) => {
            return b.updatedTs - a.updatedTs;
          }),
        });
      }
      if (state.showClosed) {
        list.push({
          title: "Closed",
          list: filteredList(state.closedList).sort((a, b) => {
            return b.updatedTs - a.updatedTs;
          }),
        });
      }
      return list;
    });

    return {
      searchField,
      state,
      sectionList,
      selectEnvironment,
      changeSearchText,
    };
  },
};
</script>
