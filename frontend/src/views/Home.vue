<template>
  <div class="flex flex-col">
    <div class="px-2 py-1">
      <EnvironmentTabFilter @select-environment="selectEnvironment" />
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
import { watchEffect, computed, reactive } from "vue";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import TaskTable from "../components/TaskTable.vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { activeStage, activeEnvironmentId } from "../utils";
import { User, Environment, Task, StageStatus } from "../types";

interface LocalState {
  attentionList: Task[];
  subscribeList: Task[];
  closeList: Task[];
  selectedEnvironment?: Environment;
}

export default {
  name: "Home",
  components: {
    EnvironmentTabFilter,
    TaskTable,
  },
  props: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({
      attentionList: [],
      subscribeList: [],
      closeList: [],
    });
    const store = useStore();
    const router = useRouter();
    const currentUser: User = computed(() =>
      store.getters["auth/currentUser"]()
    ).value;

    const prepareTaskList = () => {
      store
        .dispatch("task/fetchTaskListForUser", currentUser.id)
        .then((taskList: Task[]) => {
          state.attentionList = [];
          state.subscribeList = [];
          state.closeList = [];
          for (const task of taskList) {
            // "OPEN"
            if (task.status === "OPEN") {
              if (
                task.creator.id === currentUser.id ||
                task.assignee?.id === currentUser.id
              ) {
                state.attentionList.push(task);
              } else if (task.subscriberIdList.includes(currentUser.id)) {
                state.subscribeList.push(task);
              }
            }
            // "DONE" or "CANCELED"
            else if (task.status === "DONE" || task.status === "CANCELED") {
              if (
                task.creator.id === currentUser.id ||
                task.assignee?.id === currentUser.id
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

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
    };

    const filteredList = (list: Task[]) => {
      if (!state.selectedEnvironment) {
        // Select "All"
        return list;
      }
      return list.filter((task) => {
        if (state.selectedEnvironment) {
          return activeEnvironmentId(task) === state.selectedEnvironment.id;
        }
        return false;
      });
    };

    const openTaskSorter = (a: Task, b: Task) => {
      const statusOrder = (status: StageStatus) => {
        switch (status) {
          case "PENDING":
            return 0;
          case "FAILED":
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

    watchEffect(prepareTaskList);

    return {
      state,
      filteredList,
      selectEnvironment,
      openTaskSorter,
    };
  },
};
</script>
