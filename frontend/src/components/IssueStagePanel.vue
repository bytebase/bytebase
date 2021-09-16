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
          :tabList="tabList"
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
        :taskCheckRunList="task.taskCheckRunList"
      />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, watch, PropType, computed } from "vue";
import TaskCheckRunTable from "../components/TaskCheckRunTable.vue";
import TaskRunTable from "../components/TaskRunTable.vue";
import { Stage, Task } from "../types";
import { activeTaskInStage } from "../utils";

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

    const tabList = computed((): string[] => {
      return ["Run", "Check"];
    });

    const activeTask = computed((): Task => {
      return activeTaskInStage(props.stage);
    });

    return { RUN_TAB, CHECK_TAB, state, tabList, activeTask };
  },
};
</script>
