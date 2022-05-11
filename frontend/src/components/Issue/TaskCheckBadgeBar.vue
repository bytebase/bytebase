<template>
  <div class="flex items-center space-x-2">
    <template
      v-for="(checkRun, index) in filteredTaskCheckRunList"
      :key="index"
    >
      <button
        class="inline-flex items-center px-3 py-0.5 rounded-full text-sm border border-control-border"
        :class="buttonStyle(checkRun)"
        @click.prevent="selectTaskCheckRun(checkRun)"
      >
        <template v-if="checkRun.status == 'RUNNING'">
          <TaskSpinner class="-ml-1 mr-1.5 h-4 w-4 text-info" />
        </template>
        <template v-else-if="checkRun.status == 'DONE'">
          <template v-if="taskCheckStatus(checkRun) == 'SUCCESS'">
            <heroicons-outline:check
              class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-success"
            />
          </template>
          <template v-else-if="taskCheckStatus(checkRun) == 'WARN'">
            <heroicons-outline:exclamation
              class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-warning"
            />
          </template>
          <template v-else-if="taskCheckStatus(checkRun) == 'ERROR'">
            <span class="mr-1.5 font-medium text-error" aria-hidden="true">
              !
            </span>
          </template>
        </template>
        <template v-else-if="checkRun.status == 'FAILED'">
          <span class="mr-1.5 font-medium text-error" aria-hidden="true">
            !
          </span>
        </template>
        <template v-else-if="checkRun.status == 'CANCELED'">
          <heroicons-outline:ban
            class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-control"
          />
        </template>
        {{ name(checkRun) }}
      </button>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { TaskCheckRun, TaskCheckStatus, TaskCheckType } from "../../types";
import TaskSpinner from "./TaskSpinner.vue";

interface LocalState {
  selectedTaskCheckRun?: TaskCheckRun;
}

export default defineComponent({
  name: "TaskCheckBadgeBar",
  components: { TaskSpinner },
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
      default: undefined,
    },
  },
  emits: ["select-task-check-run"],
  setup(props, { emit }) {
    const { t } = useI18n();
    const state = reactive<LocalState>({
      selectedTaskCheckRun: props.selectedTaskCheckRun,
    });

    watch(
      () => props.selectedTaskCheckRun,
      (curNew, _) => {
        state.selectedTaskCheckRun = curNew;
      }
    );

    const buttonStyle = (checkRun: TaskCheckRun): string => {
      let bgColor = "";
      let bgHoverColor = "";
      let textColor = "";
      switch (checkRun.status) {
        case "RUNNING":
          bgColor = "bg-blue-100";
          bgHoverColor = "bg-blue-300";
          textColor = "text-blue-800";
          break;
        case "FAILED":
          bgColor = "bg-red-100";
          bgHoverColor = "bg-red-300";
          textColor = "text-red-800";
          break;
        case "CANCELED":
          bgColor = "bg-yellow-100";
          bgHoverColor = "bg-yellow-300";
          textColor = "text-yellow-800";
          break;
        case "DONE":
          switch (taskCheckStatus(checkRun)) {
            case "SUCCESS":
              bgColor = "bg-gray-100";
              bgHoverColor = "bg-gray-300";
              textColor = "text-gray-800";
              break;
            case "WARN":
              bgColor = "bg-yellow-100";
              bgHoverColor = "bg-yellow-300";
              textColor = "text-yellow-800";
              break;
            case "ERROR":
              bgColor = "bg-red-100";
              bgHoverColor = "bg-red-300";
              textColor = "text-red-800";
              break;
          }
          break;
      }

      const styleList: string[] = [textColor];
      if (props.allowSelection) {
        styleList.push("cursor-pointer", `hover:${bgHoverColor}`);
        if (
          props.stickySelection &&
          checkRun.type == state.selectedTaskCheckRun?.type
        ) {
          styleList.push(bgHoverColor);
        } else {
          styleList.push(bgColor);
        }
      } else {
        styleList.push(bgColor);
        styleList.push("cursor-default");
      }

      return styleList.join(" ");
    };

    // For a particular check type, only returns the most recent one
    const filteredTaskCheckRunList = computed((): TaskCheckRun[] => {
      const result: TaskCheckRun[] = [];
      for (const run of props.taskCheckRunList) {
        const index = result.findIndex((item) => item.type == run.type);
        if (index < 0) {
          result.push(run);
        } else if (result[index].updatedTs < run.updatedTs) {
          result[index] = run;
        }
      }

      return result.sort((a: TaskCheckRun, b: TaskCheckRun) => {
        // Put likely failure first.
        const taskCheckRunTypeOrder = (type: TaskCheckType) => {
          switch (type) {
            case "bb.task-check.general.earliest-allowed-time":
              return 0;
            case "bb.task-check.database.statement.compatibility":
              return 1;
            case "bb.task-check.database.statement.syntax":
              return 2;
            case "bb.task-check.database.connect":
              return 3;
            case "bb.task-check.instance.migration-schema":
              return 4;
            case "bb.task-check.database.statement.advise":
              return 5;
            case "bb.task-check.database.statement.fake-advise":
              return 100;
          }
        };

        return taskCheckRunTypeOrder(a.type) - taskCheckRunTypeOrder(b.type);
      });
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
          return t("task.check-type.fake");
        case "bb.task-check.database.statement.syntax":
          return t("task.check-type.syntax");
        case "bb.task-check.database.statement.compatibility":
          return t("task.check-type.compatibility");
        case "bb.task-check.database.statement.advise":
          return t("task.check-type.sql-review");
        case "bb.task-check.database.connect":
          return t("task.check-type.connection");
        case "bb.task-check.instance.migration-schema":
          return t("task.check-type.migration-schema");
        case "bb.task-check.general.earliest-allowed-time":
          return t("task.check-type.earliest-allowed-time");
      }
    };

    const selectTaskCheckRun = (taskCheckRun: TaskCheckRun) => {
      emit("select-task-check-run", taskCheckRun);
    };

    return {
      state,
      buttonStyle,
      filteredTaskCheckRunList,
      taskCheckStatus,
      name,
      selectTaskCheckRun,
    };
  },
});
</script>
