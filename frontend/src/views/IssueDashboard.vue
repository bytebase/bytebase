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
      :issueSectionList="[
        {
          title: 'Closed',
          list: filteredList(state.closedList).sort((a, b) => {
            return b.updatedTs - a.updatedTs;
          }),
        },
      ]"
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

interface LocalState {
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

    const state = reactive<LocalState>({
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

    return {
      searchField,
      state,
      filteredList,
      selectEnvironment,
      changeSearchText,
    };
  },
};
</script>
