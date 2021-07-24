<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :selectedId="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <div class="flex flex-row space-x-4">
        <MemberSelect
          v-if="scopeByPrincipal"
          :selectedId="state.selectedPrincipalId"
          @select-principal-id="selectPrincipal"
        />
        <BBTableSearch
          ref="searchField"
          :placeholder="'Search issue name'"
          @change-text="(text) => changeSearchText(text)"
        />
      </div>
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
import MemberSelect from "../components/MemberSelect.vue";
import { Environment, Issue, PrincipalId, ProjectId } from "../types";
import { computed, onMounted, watchEffect } from "@vue/runtime-core";
import { activeEnvironment } from "../utils";
import { BBTableSectionDataSource } from "../bbkit/types";

interface LocalState {
  showOpen: boolean;
  showClosed: boolean;
  openList: Issue[];
  closedList: Issue[];
  searchText: string;
  selectedPrincipalId: PrincipalId;
  selectedEnvironment?: Environment;
  selectedProjectId?: ProjectId;
}

export default {
  name: "IssueDashboard",
  components: { EnvironmentTabFilter, IssueTable, MemberSelect },
  setup(props, ctx) {
    const searchField = ref();

    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const statusList: string[] = router.currentRoute.value.query.status
      ? (router.currentRoute.value.query.status as string).split(",")
      : [];

    const state = reactive<LocalState>({
      showOpen: statusList.length == 0 || statusList.includes("open"),
      showClosed: statusList.length == 0 || statusList.includes("closed"),
      openList: [],
      closedList: [],
      searchText: "",
      selectedPrincipalId: currentUser.value.id,
      selectedEnvironment: router.currentRoute.value.query.environment
        ? store.getters["environment/environmentById"](
            router.currentRoute.value.query.environment
          )
        : undefined,
      selectedProjectId: router.currentRoute.value.query.project
        ? parseInt(router.currentRoute.value.query.project as string)
        : undefined,
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    // Applies principal scope if we explicitly specify user in the query parameter
    // or project is NOT present in the query parameter.
    // In other words, if only project is present in the query parameter, then
    // we do NOT apply principal scope, which is the case if we want to list all issues
    // for a particular project.
    const scopeByPrincipal = computed(() => {
      return router.currentRoute.value.query.user || !state.selectedProjectId;
    });

    const prepareIssueList = () => {
      // We call open and close separately because normally the number of open issues is limited
      // while the closed issues could be a lot.
      if (state.showOpen) {
        store
          .dispatch("issue/fetchIssueList", {
            issueStatusList: ["OPEN"],
            userId: scopeByPrincipal.value
              ? state.selectedPrincipalId
              : undefined,
            projectId: state.selectedProjectId,
          })
          .then((issueList: Issue[]) => {
            state.openList = issueList;
          });
      }

      if (state.showClosed) {
        store
          .dispatch("issue/fetchIssueList", {
            issueStatusList: ["DONE", "CANCELED"],
            userId: scopeByPrincipal.value
              ? state.selectedPrincipalId
              : undefined,
            projectId: state.selectedProjectId,
          })
          .then((issueList: Issue[]) => {
            state.closedList = issueList;
          });
      }
    };

    watchEffect(prepareIssueList);

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

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
      if (environment) {
        router.replace({
          name: "workspace.issue",
          query: {
            ...router.currentRoute.value.query,
            environment: environment.id,
          },
        });
      } else {
        router.replace({
          name: "workspace.issue",
          query: {
            ...router.currentRoute.value.query,
            environment: undefined,
          },
        });
      }
    };

    const selectPrincipal = (principalId: PrincipalId) => {
      state.selectedPrincipalId = principalId;
      router.replace({
        name: "workspace.issue",
        query: {
          ...router.currentRoute.value.query,
          user: principalId,
        },
      });
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    return {
      searchField,
      state,
      scopeByPrincipal,
      sectionList,
      selectEnvironment,
      selectPrincipal,
      changeSearchText,
    };
  },
};
</script>
