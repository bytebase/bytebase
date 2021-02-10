<template>
  <BBTable
    :columnList="state.columnList"
    :sectionDataSource="taskSectionList"
    :showHeader="false"
    @click-row="clickTask"
  >
    <template v-slot:body="{ rowData: task }">
      <BBTableCell :leftPadding="4" class="w-4 table-cell">
        <span
          class="w-5 h-5 flex items-center justify-center rounded-full"
          :class="iconStatusMap[task.attributes.status].class"
        >
          <template v-if="task.attributes.status == `PENDING`">
            <span
              class="h-1.5 w-1.5 bg-blue-600 hover:bg-blue-700 rounded-full"
              aria-hidden="true"
            ></span>
          </template>
          <template v-else-if="task.attributes.status == `RUNNING`">
            <span
              class="h-2 w-2 bg-blue-600 hover:bg-blue-700 rounded-full"
              style="
                animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
              "
              aria-hidden="true"
            ></span>
          </template>
          <template v-else-if="task.attributes.status == `DONE`">
            <svg
              class="w-4 h-4"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fill-rule="evenodd"
                d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                clip-rule="evenodd"
              />
            </svg>
          </template>
          <template v-else-if="task.attributes.status == `FAILED`">
            <span
              class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
              aria-hidden="true"
              >!</span
            >
          </template>
          <template v-else-if="task.attributes.status == `CANCELED`">
            <svg
              class="w-5 h-5"
              fill="currentColor"
              viewBox="0 0 20 20"
              xmlns="http://www.w3.org/2000/svg"
              aria-hidden="true"
            >
              >
              <path
                fill-rule="evenodd"
                d="M3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                clip-rule="evenodd"
              ></path>
            </svg>
          </template>
        </span>
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
        {{ environmentName(task.attributes.currentStageId) }}
      </BBTableCell>
      <BBTableCell :leftPadding="1" class="w-auto">
        {{ task.attributes.name }}
      </BBTableCell>
      <BBTableCell class="w-12 hidden sm:table-cell">
        <BBStepBar :stepList="stageList(task)" />
      </BBTableCell>
      <BBTableCell class="w-32 hidden sm:table-cell">
        {{ task.attributes.assignee.name }}
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
import { humanize, taskSlug } from "../utils";
import { EnvironmentId, Task } from "../types";

interface LocalState {
  columnList: BBTableColumn[];
  dataSource: Object[];
}

const iconStatusMap = {
  PENDING: {
    name: "Pending",
    class:
      "bg-white border-2 border-blue-600 text-blue-600 hover:text-blue-700 hover:border-blue-700",
  },
  RUNNING: {
    name: "Running",
    class:
      "bg-white border-2 border-blue-600 text-blue-600 hover:text-blue-700 hover:border-blue-700",
  },
  DONE: {
    name: "Done",
    class: "bg-success hover:bg-success-hover text-white",
  },
  FAILED: {
    name: "Failed",
    class: "bg-error text-white hover:text-white hover:bg-error-hover",
  },
  CANCELED: {
    name: "Canceled",
    class:
      "bg-white border-2 text-gray-400 border-gray-400 hover:text-gray-500 hover:border-gray-500",
  },
};

export default {
  name: "TaskTable",
  components: {},
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

    const environmentName = function (id: EnvironmentId) {
      return store.getters["environment/environmentById"](id)?.attributes.name;
    };

    const router = useRouter();

    const stageList = function (task: Task): BBStep[] {
      return task.attributes.stageProgressList.map((stageProgress) => {
        let stepStatus: BBStepStatus = "CREATED";
        switch (stageProgress.status) {
          case "CREATED":
            stepStatus = "CREATED";
            break;
          case "PENDING":
            stepStatus = "PENDING";
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
          case "CANCELED":
            stepStatus = "CANCELED";
            break;
          case "SKIPPED":
            stepStatus = "SKIPPED";
            break;
        }
        return {
          title: stageProgress.name,
          status: stepStatus,
          link: (): string => {
            return `/task/${task.id}#${stageProgress.id}`;
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
      environmentName,
      iconStatusMap,
      stageList,
      humanize,
      clickTask,
    };
  },
};
</script>
