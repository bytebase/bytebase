<template>
  <div class="flex items-center space-x-2">
    <template
      v-for="(checkRun, index) in filteredTaskCheckRunList"
      :key="index"
    >
      <button
        class="
          inline-flex
          items-center
          px-3
          py-0.5
          rounded-full
          text-sm text-main
        "
        :class="
          allowSelection
            ? stickySelection &&
              checkRun.type == state.selectedTaskCheckRun.type
              ? 'cursor-pointer border border-control-border hover:bg-control-bg-hover bg-control-bg-hover'
              : 'cursor-pointer border border-control-border hover:bg-control-bg-hover'
            : 'cursor-default'
        "
        @click.prevent="selectTaskCheckRun(checkRun)"
      >
        <template v-if="checkRun.status == 'RUNNING'">
          <svg
            class="animate-spin -ml-1 mr-1.5 h-4 w-4 text-info"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
        </template>
        <template v-else-if="checkRun.status == 'DONE'">
          <template v-if="taskCheckStatus(checkRun) == 'SUCCESS'">
            <svg
              class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-success"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M5 13l4 4L19 7"
              ></path>
            </svg>
          </template>
          <template v-else-if="taskCheckStatus(checkRun) == 'WARN'">
            <svg
              class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-warning"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
              ></path>
            </svg>
          </template>
          <template v-else-if="taskCheckStatus(checkRun) == 'ERROR'">
            <span class="mr-1.5 font-medium text-error" aria-hidden="true"
              >!</span
            >
          </template>
        </template>
        <template v-else-if="checkRun.status == 'FAILED'">
          <span class="mr-1.5 font-medium text-error" aria-hidden="true"
            >!</span
          >
        </template>
        <template v-else-if="checkRun.status == 'CANCELED'">
          <svg
            class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-control"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
            ></path>
          </svg>
        </template>
        {{ name(checkRun) }}
      </button>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, PropType, reactive, watch } from "vue";
import { TaskCheckRun, TaskCheckStatus, TaskCheckType } from "../types";

interface LocalState {
  selectedTaskCheckRun?: TaskCheckRun;
}

export default {
  name: "TaskCheckBadgeBar",
  emits: ["select-task-check-run"],
  props: {
    taskCheckRunList: {
      required: true,
      type: Object as PropType<TaskCheckRun[]>,
    },
    allowSelection: {
      default: true,
      type: Boolean,
    },
    stickySelection: {
      default: false,
      type: Boolean,
    },
    selectedTaskCheckRun: {
      type: Object as PropType<TaskCheckRun>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      selectedTaskCheckRun: props.selectedTaskCheckRun,
    });

    watch(
      () => props.selectedTaskCheckRun,
      (curNew, _) => {
        state.selectedTaskCheckRun = curNew;
      }
    );

    // For a particular check type, only returns the most recent one
    const filteredTaskCheckRunList = computed((): TaskCheckRun[] => {
      const result: TaskCheckRun[] = [];
      for (const run of props.taskCheckRunList) {
        const index = result.findIndex((item) => item.type == run.type);
        if (index >= 0 && result[index].updatedTs < run.updatedTs) {
          result[index] = run;
        } else {
          result.push(run);
        }
      }
      return result;
    });

    // Returns the most severe status
    const taskCheckStatus = (taskCheckRun: TaskCheckRun): TaskCheckStatus => {
      let value: TaskCheckStatus = "SUCCESS";
      for (const result of taskCheckRun.result.resultList) {
        if (result.status == "ERROR") {
          return "ERROR";
        }
        if (result.status == "WARN") {
          value = "WARN";
        }
      }
      return value;
    };

    const name = (taskCheckRun: TaskCheckRun): string => {
      switch (taskCheckRun.type) {
        case "bb.task-check.database.statement.fake-advise":
          return "Fake";
        case "bb.task-check.database.statement.syntax":
          return "Syntax";
        case "bb.task-check.database.connect":
          return "Connection";
      }
    };

    const selectTaskCheckRun = (taskCheckRun: TaskCheckRun) => {
      emit("select-task-check-run", taskCheckRun);
    };

    return {
      state,
      filteredTaskCheckRunList,
      taskCheckStatus,
      name,
      selectTaskCheckRun,
    };
  },
};
</script>
