<template>
  <div class="flex items-start space-x-4">
    <button
      v-if="showRunCheckButton"
      type="button"
      class="btn-small py-0.5"
      :disabled="hasRunningTaskCheck"
      @click.prevent="runChecks"
    >
      {{ hasRunningTaskCheck ? $t("task.checking") : $t("task.run-task") }}
    </button>
    <TaskCheckBadgeBar
      :task-check-run-list="task.taskCheckRunList"
      @select-task-check-type="viewCheckRunDetail"
    />
    <BBModal
      v-if="state.showModal"
      :title="$t('task.check-result.title', { name: task.name })"
      @close="dismissDialog"
    >
      <div class="space-y-4 w-208">
        <div>
          <TaskCheckBadgeBar
            :task-check-run-list="task.taskCheckRunList"
            :allow-selection="true"
            :sticky-selection="true"
            :selected-task-check-type="state.selectedTaskCheckType"
            @select-task-check-type="viewCheckRunDetail"
          />
        </div>
        <BBTabFilter
          class="pt-4"
          :tab-item-list="tabItemList"
          :selected-index="state.selectedTabIndex"
          @select-index="
            (index: number) => {
              state.selectedTabIndex = index;
            }
          "
        />
        <TaskCheckRunPanel
          v-if="selectedTaskCheckRun"
          :task-check-run="selectedTaskCheckRun"
        />
        <div class="pt-4 flex justify-end">
          <button
            type="button"
            class="btn-primary py-2 px-4"
            @click.prevent="dismissDialog"
          >
            {{ $t("common.close") }}
          </button>
        </div>
      </div>
    </BBModal>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { cloneDeep } from "lodash-es";
import { Task, TaskCheckRun, TaskCheckStatus, TaskCheckType } from "@/types";
import TaskCheckBadgeBar from "./TaskCheckBadgeBar.vue";
import TaskCheckRunPanel from "./TaskCheckRunPanel.vue";
import { BBTabFilterItem } from "@/bbkit/types";
import { humanizeTs } from "@/utils";

interface LocalState {
  showModal: boolean;
  selectedTaskCheckType: TaskCheckType | undefined;
  selectedTabIndex: number;
}

export default defineComponent({
  name: "TaskCheckBar",
  components: { TaskCheckBadgeBar, TaskCheckRunPanel },
  props: {
    allowRunTask: {
      type: Boolean,
      default: true,
    },
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  emits: ["run-checks"],
  setup(props, { emit }) {
    const { t } = useI18n();

    const state = reactive<LocalState>({
      showModal: false,
      selectedTaskCheckType: undefined,
      selectedTabIndex: 0,
    });

    const tabTaskCheckRunList = computed((): TaskCheckRun[] => {
      if (!state.selectedTaskCheckType) {
        return [];
      }

      const list: TaskCheckRun[] = [];
      for (const check of props.task.taskCheckRunList) {
        if (check.type == state.selectedTaskCheckType) {
          list.push(check);
        }
      }
      const clonedList = cloneDeep(list);
      clonedList.sort(
        (a: TaskCheckRun, b: TaskCheckRun) => b.createdTs - a.createdTs
      );
      return clonedList;
    });

    const tabItemList = computed((): BBTabFilterItem[] => {
      return tabTaskCheckRunList.value.map((item, index) => {
        return {
          title: index == 0 ? t("common.latest") : humanizeTs(item.createdTs),
          alert: false,
        };
      });
    });

    const selectedTaskCheckRun = computed(() => {
      const type = state.selectedTaskCheckType;
      const index = state.selectedTabIndex;
      if (!type) return undefined;
      return tabTaskCheckRunList.value[index];
    });

    const showRunCheckButton = computed((): boolean => {
      if (!props.allowRunTask) return false;
      return (
        (props.task.type == "bb.task.database.schema.update" ||
          props.task.type == "bb.task.database.data.update") &&
        (props.task.status == "PENDING" ||
          props.task.status == "PENDING_APPROVAL" ||
          props.task.status == "RUNNING" ||
          props.task.status == "FAILED")
      );
    });

    const hasRunningTaskCheck = computed((): boolean => {
      for (const check of props.task.taskCheckRunList) {
        if (check.status == "RUNNING") {
          return true;
        }
      }
      return false;
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

    const viewCheckRunDetail = (type: TaskCheckType) => {
      state.selectedTaskCheckType = type;
      state.selectedTabIndex = 0;
      state.showModal = true;
    };

    const dismissDialog = () => {
      state.showModal = false;
      state.selectedTaskCheckType = undefined;
    };

    const runChecks = () => {
      emit("run-checks", props.task);
    };

    return {
      state,
      tabTaskCheckRunList,
      tabItemList,
      selectedTaskCheckRun,
      showRunCheckButton,
      hasRunningTaskCheck,
      taskCheckStatus,
      viewCheckRunDetail,
      dismissDialog,
      runChecks,
    };
  },
});
</script>
