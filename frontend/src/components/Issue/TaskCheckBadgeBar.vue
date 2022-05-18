<template>
  <div class="flex items-center flex-wrap gap-2 flex-1">
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
import { groupBy, maxBy } from "lodash-es";
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
      const groupByType = groupBy(props.taskCheckRunList, (run) => run.type);
      /*
        `groupByType` looks like: {
          "bb.task-check.general.earliest-allowed-time": [run1, run2, ...],
          "bb.task-check.database.statement.compatibility": [run1, run2, ...],
          "bb.task-check.database.statement.syntax": [run1, run2, ...],
          ...
        }
        `result` is an array of the most recent TaskCheckRun in each group
      */
      const result = Object.keys(groupByType).map((type) => {
        const groupList = groupByType[type];
        const mostRecentInGroup = maxBy(groupList, (run) => run.updatedTs)!;
        return mostRecentInGroup;
      });

      return result.sort((a: TaskCheckRun, b: TaskCheckRun) => {
        const taskCheckRunTypeOrder = (type: TaskCheckType) => {
          const has = TaskCheckTypeOrderDict.has(type);
          console.assert(has, `Missing TaskCheckType order of "${type}"`);
          if (has) {
            return TaskCheckTypeOrderDict.get(type)!;
          }
          // Fallback, types not defined in the dictionary will go to the tail.
          return FAKE_MAX_TASK_CHECK_TYPE_ORDER;
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
      const { type } = taskCheckRun;
      const has = TaskCheckTypeNameDict.has(type);
      console.assert(has, `Missing TaskCheckType name of "${type}"`);
      if (has) {
        const key = TaskCheckTypeNameDict.get(type)!;
        return t(key);
      }
      return type;
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

// Defines the order of TaskCheckType
const TaskCheckTypeOrderList: TaskCheckType[] = [
  "bb.task-check.general.earliest-allowed-time",
  "bb.task-check.database.statement.compatibility",
  "bb.task-check.database.statement.syntax",
  "bb.task-check.database.connect",
  "bb.task-check.instance.migration-schema",
  "bb.task-check.database.statement.advise",
];
const TaskCheckTypeOrderDict = new Map<TaskCheckType, number>(
  TaskCheckTypeOrderList.map((type, index) => [type, index])
);
const FAKE_MAX_TASK_CHECK_TYPE_ORDER = 100;
TaskCheckTypeOrderDict.set(
  "bb.task-check.database.statement.fake-advise",
  FAKE_MAX_TASK_CHECK_TYPE_ORDER
);

// Defines the mapping from TaskCheckType to an i18n resource keypath
const TaskCheckTypeNameDict = new Map<TaskCheckType, string>([
  ["bb.task-check.database.statement.fake-advise", "task.check-type.fake"],
  ["bb.task-check.database.statement.syntax", "task.check-type.syntax"],
  [
    "bb.task-check.database.statement.compatibility",
    "task.check-type.compatibility",
  ],
  ["bb.task-check.database.statement.advise", "task.check-type.sql-review"],
  ["bb.task-check.database.connect", "task.check-type.connection"],
  [
    "bb.task-check.instance.migration-schema",
    "task.check-type.migration-schema",
  ],
  [
    "bb.task-check.general.earliest-allowed-time",
    "task.check-type.earliest-allowed-time",
  ],
]);
</script>
