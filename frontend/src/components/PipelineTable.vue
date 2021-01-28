<template>
  <BBTable
    :columnList="state.columnList"
    :sectionDataSource="pipelineSectionList"
    :showHeader="false"
    @click-row="clickPipeline"
  >
    <template v-slot:body="{ rowData: pipeline }">
      <BBTableCell class="w-1/12 hidden lg:table-cell">
        <span class="">#{{ pipeline.id }}</span>
      </BBTableCell>
      <BBTableCell class="w-1/12">
        <span class="">{{ pipeline.attributes.type }}</span>
      </BBTableCell>
      <BBTableCell class="w-1/12">
        <span
          class="px-2 inline-flex text-xs font-semibold rounded-full"
          :class="statusMap[pipeline.attributes.status].class"
        >
          {{ statusMap[pipeline.attributes.status].name }}
        </span>
      </BBTableCell>
      <BBTableCell class="w-5/12">
        {{ pipeline.attributes.name }}
      </BBTableCell>
      <BBTableCell class="w-2/12">
        <BBStepBar :stepList="stageList(pipeline)" />
      </BBTableCell>
      <BBTableCell class="w-1/12 hidden lg:table-cell">
        <BBAvatar :username="pipeline.attributes.assignee.name"> </BBAvatar>
      </BBTableCell>
      <BBTableCell class="w-1/12 hidden lg:table-cell">
        {{ humanize(pipeline.attributes.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useRouter } from "vue-router";
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
  CREATED: {
    name: "Created",
    class: "bg-green-200 text-green-800",
  },
  RUNNING: {
    name: "Running",
    class: "bg-yellow-200 text-yellow-800",
  },
  DONE: {
    name: "Done",
    class: "bg-blue-200 text-blue-800",
  },
  FAILED: {
    name: "Failed",
    class: "bg-red-200 text-red-800",
  },
  CANCELED: {
    name: "Canceled",
    class: "bg-gray-200 text-gray-800",
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
          title: "ID",
        },
        {
          title: "Type",
        },
        {
          title: "Status",
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
          title: "Created",
        },
      ],
      dataSource: [],
    });

    const router = useRouter();

    const stageList = function (pipeline: Pipeline): BBStep[] {
      return pipeline.attributes.stageProgressList.map((stageProgress) => {
        let stepStatus: BBStepStatus = "CREATED";
        switch (stageProgress.status) {
          case "RUNNING":
            stepStatus = "RUNNING";
            break;
          case "DONE":
            stepStatus = "DONE";
            break;
          case "FAILED":
            stepStatus = "FAILED";
            break;
          case "CREATED":
            stepStatus = "CREATED";
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
      clickPipeline,
    };
  },
};
</script>
