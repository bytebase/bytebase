<template>
  <BBTable
    :columnList="columnList"
    :dataSource="taskRunList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    :rowClickable="false"
  >
    <template v-slot:body="{ rowData: taskRun }">
      <BBTableCell :leftPadding="4" class="table-cell w-12">
        <div class="flex flex-row space-x-2">
          <div
            class="
              relative
              w-5
              h-5
              flex flex-shrink-0
              items-center
              justify-center
              rounded-full
              select-none
            "
            :class="statusIconClass(taskRun.status)"
          >
            <template v-if="taskRun.status == 'RUNNING'">
              <span
                class="h-2.5 w-2.5 bg-info rounded-full"
                style="
                  animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
                "
                aria-hidden="true"
              ></span>
            </template>
            <template v-else-if="taskRun.status == 'DONE'">
              <svg
                class="w-5 h-5"
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
            <template v-else-if="taskRun.status == 'FAILED'">
              <span class="text-white font-medium text-base" aria-hidden="true"
                >!</span
              >
            </template>
            <template v-else-if="taskRun.status == 'CANCELED'">
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
                ></path></svg
            ></template>
          </div>
          <div class="flex items-center capitalize">
            {{ taskRun.status.toLowerCase() }}
          </div>
        </div>
      </BBTableCell>
      <BBTableCell class="table-cell w-36">
        {{ taskRun.comment }}
      </BBTableCell>
      <BBTableCell class="table-cell w-12">
        <div class="flex flex-row items-center space-x-2">
          <PrincipalAvatar :principal="taskRun.creator" :size="'SMALL'" />
          <div class="flex flex-col">
            <div class="flex flex-row items-center space-x-2">
              <router-link :to="`/u/${taskRun.creator.id}`"
                >{{ taskRun.creator.name }}
              </router-link>
            </div>
          </div>
        </div>
      </BBTableCell>
      <BBTableCell class="table-cell w-12">
        {{ humanizeTs(taskRun.createdTs) }}
      </BBTableCell>
      <BBTableCell class="table-cell w-12">
        {{ humanizeTs(taskRun.updatedTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import PrincipalAvatar from "./PrincipalAvatar.vue";
import { BBTableColumn } from "../bbkit/types";
import { TaskRun, TaskRunStatus } from "../types";

const columnList: BBTableColumn[] = [
  {
    title: "Status",
  },
  {
    title: "Comment",
  },
  {
    title: "Invoker",
  },
  {
    title: "Started",
  },
  {
    title: "Ended",
  },
];

export default {
  name: "TaskRunTable",
  components: { PrincipalAvatar },
  props: {
    taskRunList: {
      required: true,
      type: Object as PropType<TaskRun[]>,
    },
  },
  setup(props, ctx) {
    const statusIconClass = (status: TaskRunStatus) => {
      switch (status) {
        case "RUNNING":
          return "bg-white border-2 border-info text-info";
        case "DONE":
          return "bg-success text-white";
        case "FAILED":
          return "bg-error text-white";
        case "CANCELED":
          return "bg-white border-2 border-gray-400 text-gray-400";
      }
    };

    return {
      columnList,
      statusIconClass,
    };
  },
};
</script>
