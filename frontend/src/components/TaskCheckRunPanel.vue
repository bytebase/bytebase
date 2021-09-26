<template>
  <div>
    <BBTable
      :columnList="columnList"
      :dataSource="checkResultList"
      :showHeader="false"
      :leftBordered="true"
      :rightBordered="true"
      :topBordered="true"
      :bottomBordered="true"
      :rowClickable="false"
    >
      <template v-slot:body="{ rowData: checkResult }">
        <BBTableCell :leftPadding="4" class="table-cell w-4">
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
              :class="statusIconClass(checkResult.status)"
            >
              <template v-if="checkResult.status == 'SUCCESS'">
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
              <template v-if="checkResult.status == 'WARN'">
                <svg
                  class="h-4 w-4"
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  aria-hidden="true"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                  ></path>
                </svg>
              </template>
              <template v-else-if="checkResult.status == 'ERROR'">
                <span
                  class="text-white font-medium text-base"
                  aria-hidden="true"
                  >!</span
                >
              </template>
            </div>
          </div>
        </BBTableCell>
        <BBTableCell class="table-cell w-16">
          {{ checkResult.title }}
        </BBTableCell>
        <BBTableCell class="table-cell w-48">
          {{ checkResult.content }}
        </BBTableCell>
      </template>
    </BBTable>
  </div>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { TaskCheckStatus, TaskCheckRun, TaskCheckResult } from "../types";

const columnList: BBTableColumn[] = [
  {
    title: "",
  },
  {
    title: "Title",
  },
  {
    title: "Detail",
  },
];

export default {
  name: "TaskCheckRunPanel",
  components: {},
  props: {
    taskCheckRun: {
      required: true,
      type: Object as PropType<TaskCheckRun>,
    },
  },
  setup(props, ctx) {
    const statusIconClass = (status: TaskCheckStatus) => {
      switch (status) {
        case "SUCCESS":
          return "bg-success text-white";
        case "WARN":
          return "bg-warning text-white";
        case "ERROR":
          return "bg-error text-white";
      }
    };

    const checkResultList = computed((): TaskCheckResult[] => {
      if (props.taskCheckRun.status == "DONE") {
        return props.taskCheckRun.result.resultList;
      } else if (props.taskCheckRun.status == "FAILED") {
        return [
          {
            status: "ERROR",
            title: "Error",
            code: props.taskCheckRun.code,
            content: props.taskCheckRun.comment,
          },
        ];
      } else if (props.taskCheckRun.status == "CANCELED") {
        return [
          {
            status: "WARN",
            title: "Canceled",
            code: props.taskCheckRun.code,
            content: "",
          },
        ];
      }

      return [];
    });

    return {
      columnList,
      statusIconClass,
      checkResultList,
    };
  },
};
</script>
