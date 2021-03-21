<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-between items-center">
      <EnvironmentTabFilter @select-environment="selectEnvironment" />
      <BBTableSearch
        ref="searchField"
        :placeholder="'Search task name'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <TaskTable
      :taskSectionList="[
        {
          title: 'Attention',
          list: filteredList(state.attentionList).sort(openTaskSorter),
        },
        {
          title: 'Subscribed',
          list: filteredList(state.subscribeList).sort(openTaskSorter),
        },
        {
          title: 'Recently Closed',
          list: filteredList(state.closeList).sort((a, b) => {
            return b.lastUpdatedTs - a.lastUpdatedTs;
          }),
        },
      ]"
    />
  </div>
</template>

<script lang="ts">
import { watchEffect, computed, nextTick, onMounted, reactive, ref } from "vue";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import TaskTable from "../components/TaskTable.vue";
import { useStore } from "vuex";
import { activeStage, activeEnvironmentId } from "../utils";
import { Environment, Task, StageStatus } from "../types";

interface LocalState {
  attentionList: Task[];
  subscribeList: Task[];
  closeList: Task[];
  selectedEnvironment?: Environment;
  searchText: string;
}

export default {
  name: "Home",
  components: {
    EnvironmentTabFilter,
    TaskTable,
  },
  props: {},
  setup(props, ctx) {
    const searchField = ref();

    const state = reactive<LocalState>({
      attentionList: [],
      subscribeList: [],
      closeList: [],
      searchText: "",
    });
    const store = useStore();
    const currentUser = computed(() => store.getters["auth/currentUser"]());

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    const prepareTaskList = () => {
      store
        .dispatch("task/fetchTaskListForUser", currentUser.value.id)
        .then((taskList: Task[]) => {
          state.attentionList = [];
          state.subscribeList = [];
          state.closeList = [];
          for (const task of taskList) {
            // "OPEN"
            if (task.status === "OPEN") {
              if (
                task.creator.id === currentUser.value.id ||
                task.assignee?.id === currentUser.value.id
              ) {
                state.attentionList.push(task);
              } else if (task.subscriberIdList.includes(currentUser.value.id)) {
                state.subscribeList.push(task);
              }
            }
            // "DONE" or "CANCELED"
            else if (task.status === "DONE" || task.status === "CANCELED") {
              if (
                task.creator.id === currentUser.value.id ||
                task.assignee?.id === currentUser.value.id
              ) {
                state.closeList.push(task);
              }
            }
          }
        })
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareTaskList);

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const filteredList = (list: Task[]) => {
      if (!state.selectedEnvironment && !state.searchText) {
        // Select "All"
        return list;
      }
      return list.filter((task) => {
        return (
          (!state.selectedEnvironment ||
            activeEnvironmentId(task) === state.selectedEnvironment.id) &&
          (!state.searchText ||
            task.name.toLowerCase().includes(state.searchText.toLowerCase()))
        );
      });
    };

    const openTaskSorter = (a: Task, b: Task) => {
      const statusOrder = (status: StageStatus) => {
        switch (status) {
          case "FAILED":
            return 0;
          case "PENDING":
            return 1;
          case "RUNNING":
            return 2;
          case "DONE":
            return 3;
          case "SKIPPED":
            return 4;
        }
      };
      const aStatusOrder = statusOrder(activeStage(a).status);
      const bStatusOrder = statusOrder(activeStage(b).status);
      if (aStatusOrder == bStatusOrder) {
        return b.lastUpdatedTs - a.lastUpdatedTs;
      }
      return aStatusOrder - bStatusOrder;
    };

    return {
      searchField,
      state,
      filteredList,
      selectEnvironment,
      changeSearchText,
      openTaskSorter,
    };
  },
};
</script>
