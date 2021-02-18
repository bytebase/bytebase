<template>
  <div class="flex flex-col">
    <div class="px-2 py-1">
      <EnvironmentTabFilter @select-environment="selectEnvironment" />
    </div>
    <TaskTable
      :taskSectionList="[
        {
          title: 'Attention',
          list: filteredList(state.attentionList).sort((a, b) => {
            return b.attributes.lastUpdatedTs - a.attributes.lastUpdatedTs;
          }),
        },
        {
          title: 'Subscribed',
          list: filteredList(state.subscribeList).sort((a, b) => {
            return b.attributes.lastUpdatedTs - a.attributes.lastUpdatedTs;
          }),
        },
        {
          title: 'Recently Closed',
          list: filteredList(state.closeList).sort((a, b) => {
            return b.attributes.lastUpdatedTs - a.attributes.lastUpdatedTs;
          }),
        },
      ]"
    />
  </div>
</template>

<script lang="ts">
import { watchEffect, inject, reactive } from "vue";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import TaskTable from "../components/TaskTable.vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { UserStateSymbol } from "../components/ProvideUser.vue";
import { activeStage } from "../utils";
import { User, Environment, Task } from "../types";

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
    const currentUser = inject<User>(UserStateSymbol);

    const prepareTaskList = () => {
      store
        .dispatch("task/fetchTaskListForUser", currentUser!.id)
        .then((taskList: Task[]) => {
          state.attentionList = [];
          state.subscribeList = [];
          state.closeList = [];
          for (const task of taskList) {
            if (
              task.attributes.assignee?.id === currentUser!.id &&
              task.attributes.status === "OPEN"
            ) {
              state.attentionList.push(task);
            } else if (
              task.attributes.subscriberIdList.includes(currentUser!.id) &&
              task.attributes.status === "OPEN"
            ) {
              state.subscribeList.push(task);
            } else if (
              (task.attributes.creator.id === currentUser!.id ||
                task.attributes.assignee?.id === currentUser!.id) &&
              (task.attributes.status === "DONE" ||
                task.attributes.status === "CANCELED")
            ) {
              state.closeList.push(task);
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
          const stage = activeStage(task);
          return (
            stage.type === "ENVIRONMENT" &&
            stage.environmentId == state.selectedEnvironment.id
          );
        }
        return false;
      });
    };

    watchEffect(prepareTaskList);

    return {
      state,
      filteredList,
      selectEnvironment,
    };
  },
};
</script>
