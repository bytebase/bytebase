<template>
  <BBTable
    :columnList="state.columnList"
    :sectionDataSource="pipelineSectionList"
    :showHeader="false"
    @click-row="clickPipeline"
  >
    <template v-slot:body="{ rowData: pipeline }">
      <BBTableCell :leftPadding="4" class="w-4 table-cell">
        <span
          class="w-5 h-5 flex items-center justify-center rounded-full"
          :class="statusMap[pipeline.attributes.status].class"
        >
          <template v-if="pipeline.attributes.status == `PENDING`">
            <span
              class="h-2 w-2 bg-blue-600 hover:bg-blue-700 rounded-full"
              aria-hidden="true"
            ></span>
          </template>
          <template v-else-if="pipeline.attributes.status == `RUNNING`">
            <span
              class="h-2 w-2 bg-blue-600 hover:bg-blue-700 rounded-full"
              style="
                animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
              "
              aria-hidden="true"
            ></span>
          </template>
          <template v-else-if="pipeline.attributes.status == `DONE`">
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
          <template v-else-if="pipeline.attributes.status == `FAILED`">
            <span
              class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
              aria-hidden="true"
              >!</span
            >
          </template>
          <template v-else-if="pipeline.attributes.status == `CANCELED`">
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
        <span class="">#{{ pipeline.id }}</span>
      </BBTableCell>
      <BBTableCell :rightPadding="0" class="w-4">
        <span
          class="border border-gray-500 px-1 text-gray-600 text-xs font-semibold"
        >
          {{
            pipeline.attributes.type == "AUTH"
              ? capitalize(pipeline.attributes.type)
              : pipeline.attributes.type
          }}</span
        >
      </BBTableCell>
      <BBTableCell :leftPadding="1" class="w-auto">
        {{ pipeline.attributes.name }}
      </BBTableCell>
      <BBTableCell class="w-12 hidden sm:table-cell">
        <BBStepBar :stepList="stageList(pipeline)" />
      </BBTableCell>
      <BBTableCell class="w-40 hidden sm:table-cell">
        {{ pipeline.attributes.assignee.name }}
      </BBTableCell>
      <BBTableCell :rightPadding="4" class="w-24 hidden md:table-cell">
        {{ humanize(pipeline.attributes.lastUpdatedTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useRouter } from "vue-router";
import capitalize from "lodash-es/capitalize";
import moment from "moment";
import {
  BBTableColumn,
  BBTableSectionDataSource,
  BBStep,
  BBStepStatus,
} from "../bbkit/types";
import { Pipeline } from "../types";

interface LocalState {
  columnList: BBTableColumn[];
  dataSource: Object[];
}

const statusMap = {
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
    class: "bg-accent hover:bg-accent-hover text-white",
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
  name: "PipelineTable",
  components: {},
  props: {
    pipelineSectionList: {
      required: true,
      type: Object as PropType<BBTableSectionDataSource<Pipeline>[]>,
    },
  },
  setup(props, ctx) {
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

    const router = useRouter();

    const stageList = function (pipeline: Pipeline): BBStep[] {
      return pipeline.attributes.stageProgressList.map((stageProgress) => {
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
          title: stageProgress.stageName,
          status: stepStatus,
          link: (): string => {
            return `/pipeline/${pipeline.id}#${stageProgress.stageId}`;
          },
        };
      });
    };

    const humanize = function (ts: number) {
      const time = moment.utc(ts);
      if (moment().year() == time.year()) {
        if (moment().dayOfYear() == time.dayOfYear()) {
          return time.format("HH:mm");
        }
        return time.format("MMM D");
      }
      return time.format("MMM D YYYY");
    };

    const clickPipeline = function (section: number, row: number) {
      router.push(
        `/pipeline/${props.pipelineSectionList[section].list[row].id}`
      );
    };

    return {
      state,
      statusMap,
      stageList,
      humanize,
      capitalize,
      clickPipeline,
    };
  },
};
</script>
