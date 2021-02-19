<template>
  <BBTable
    :columnList="state.columnList"
    :sectionDataSource="taskSectionList"
    :showHeader="false"
    @click-row="clickTask"
  >
    <template v-slot:body="{ rowData: task }">
      <BBTableCell :leftPadding="4" class="w-4 table-cell">
        <TaskStatusIcon :task="task" />
      </BBTableCell>

      <BBTableCell class="w-4 table-cell text-gray-500">
        <span class="">#{{ task.id }}</span>
      </BBTableCell>
      <BBTableCell :rightPadding="1" class="w-4">
        <span
          class="flex items-center justify-center px-1.5 py-0.5 rounded-full text-xs font-mono bg-gray-500 text-white"
        >
          {{ task.attributes.category }}
        </span>
      </BBTableCell>
      <BBTableCell class="w-24 table-cell">
        {{ activeEnvironmentName(task) }}
      </BBTableCell>
      <BBTableCell :leftPadding="1" class="w-auto">
        {{ task.attributes.name }}
      </BBTableCell>
      <BBTableCell class="w-12 hidden sm:table-cell">
        <BBStepBar :stepList="stageList(task)" />
      </BBTableCell>
      <BBTableCell class="w-32 hidden sm:table-cell">
        {{
          task.attributes.assignee
            ? task.attributes.assignee.name
            : "Unassigned"
        }}
      </BBTableCell>
      <BBTableCell :rightPadding="4" class="w-32 hidden md:table-cell">
        {{ humanize(task.attributes.lastUpdatedTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import {
  BBTableColumn,
  BBTableSectionDataSource,
  BBStep,
  BBStepStatus,
} from "../bbkit/types";
import TaskStatusIcon from "../components/TaskStatusIcon.vue";
import { humanize, taskSlug, activeEnvironmentId, activeStage } from "../utils";
import { EnvironmentId, Task } from "../types";

interface LocalState {
  columnList: BBTableColumn[];
  dataSource: Object[];
}

export default {
  name: "TaskTable",
  components: { TaskStatusIcon },
  props: {
    taskSectionList: {
      required: true,
      type: Object as PropType<BBTableSectionDataSource<Task>[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      columnList: [
        {
          title: "Status",
        },
        {
          title: "ID",
        },
        {
          title: "Type",
        },
        {
          title: "Environment",
        },
        {
          title: "Title",
        },
        {
          title: "Progress",
        },
        {
          title: "Assignee",
        },
        {
          title: "Updated",
        },
      ],
      dataSource: [],
    });

    const activeEnvironmentName = function (task: Task) {
      const id = activeEnvironmentId(task);
      if (id) {
        return store.getters["environment/environmentById"](id)?.attributes
          .name;
      }
      return "";
    };

    const router = useRouter();

    const stageList = function (task: Task): BBStep[] {
      return task.attributes.stageProgressList.map((stageProgress) => {
        let stepStatus: BBStepStatus = "PENDING";
        switch (stageProgress.status) {
          case "PENDING":
            if (activeStage(task).id === stageProgress.id) {
              stepStatus = "PENDING_ACTIVE";
            } else {
              stepStatus = "PENDING";
            }
            break;
          case "RUNNING":
            stepStatus = "RUNNING";
            break;
          case "DONE":
            stepStatus = "DONE";
            break;
          case "FAILED":
            stepStatus = "FAILED";
            break;
          case "SKIPPED":
            stepStatus = "SKIPPED";
            break;
        }
        return {
          title: stageProgress.name,
          status: stepStatus,
          link: (): string => {
            return `/task/${task.id}`;
          },
        };
      });
    };

    const clickTask = function (section: number, row: number) {
      const task = props.taskSectionList[section].list[row];
      router.push(`/task/${taskSlug(task.attributes.name, task.id)}`);
    };

    return {
      state,
      activeEnvironmentName,
      stageList,
      humanize,
      activeStage,
      clickTask,
    };
  },
};
</script>
