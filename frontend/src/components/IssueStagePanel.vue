<template>
  <div class="space-y-4">
    <div v-for="(task, index) in stage.taskList" :key="index">
      <div class="flex flex-row items-center space-x-1">
        <svg
          v-if="stage.taskList.length > 1 && activeTask.id == task.id"
          class="w-5 h-5 text-info"
          fill="currentColor"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            fill-rule="evenodd"
            d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z"
            clip-rule="evenodd"
          ></path>
        </svg>
        <div v-if="stage.taskList.length > 1" class="textlabel">
          <span v-if="stage.taskList.length > 1"> Step {{ index + 1 }} - </span>
          {{ task.name }}
        </div>
      </div>
      <div class="mb-2">
        <BBTabFilter
          :tabItemList="tabItemList(task)"
          :selectedIndex="state.selectedIndex"
          @select-index="
            (index) => {
              state.selectedIndex = index;
            }
          "
        />
      </div>
      <TaskRunTable
        v-if="state.selectedIndex == RUN_TAB"
        :taskRunList="task.taskRunList"
      />
      <TaskCheckRunTable
        v-else-if="state.selectedIndex == CHECK_TAB"
        :taskCheckRunList="filteredTaskCheckRunList(task.taskCheckRunList)"
      />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, watch, PropType, computed } from "vue";
import TaskCheckRunTable from "../components/TaskCheckRunTable.vue";
import TaskRunTable from "../components/TaskRunTable.vue";
import { Stage, Task, TaskCheckRun, TaskCheckType } from "../types";
import { activeTaskInStage } from "../utils";
import { BBTabFilterItem } from "../bbkit/types";

const RUN_TAB = 0;
const CHECK_TAB = 1;

interface LocalState {
  selectedIndex: number;
}

export default {
  name: "IssueStagePanel",
  props: {
    stage: {
      required: true,
      type: Object as PropType<Stage>,
    },
  },
  components: { TaskCheckRunTable, TaskRunTable },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      selectedIndex: RUN_TAB,
    });

    watch(
      () => props.stage,
      (curStage, _) => {}
    );

    // For a particular check type, only returns the most recent one
    const filteredTaskCheckRunList = (list: TaskCheckRun[]): TaskCheckRun[] => {
      const result: TaskCheckRun[] = [];
      for (const run of list) {
        const index = result.findIndex((item) => item.type == run.type);
        if (index >= 0 && result[index].updatedTs < run.updatedTs) {
          result[index] = run;
        } else {
          result.push(run);
        }
      }
      return result;
    };

    const tabItemList = (task: Task): BBTabFilterItem[] => {
      let showRunAlert = false;
      let showCheckAlert = false;

      for (const run of task.taskRunList) {
        if (run.status == "FAILED") {
          showRunAlert = true;
          break;
        }
      }
      for (const run of filteredTaskCheckRunList(task.taskCheckRunList)) {
        for (const result of run.result.resultList) {
          if (result.status == "ERROR") {
            showCheckAlert = true;
            break;
          }
        }
        if (showCheckAlert) {
          break;
        }
      }
      return [
        { title: "Runs", alert: showRunAlert },
        { title: "Checks", alert: showCheckAlert },
      ];
    };

    const activeTask = computed((): Task => {
      return activeTaskInStage(props.stage);
    });

    return {
      RUN_TAB,
      CHECK_TAB,
      state,
      filteredTaskCheckRunList,
      tabItemList,
      activeTask,
    };
  },
};
</script>
