<template>
  <nav aria-label="Progress">
    <ol
      class="border-t border-b border-block-border divide-y divide-gray-300 md:flex md:divide-y-0"
    >
      <li
        v-for="(stage, index) in stageList"
        :key="index"
        class="relative md:flex-1 md:flex"
      >
        <div class="cursor-default group flex items-center w-full">
          <span class="px-4 py-3 flex items-center text-sm font-medium">
            <div
              class="relative w-6 h-6 flex items-center justify-center rounded-full select-none"
              :class="stageIconClass(stage)"
            >
              <template v-if="stage.status === 'PENDING'">
                <span
                  v-if="activeStage(task).id === stage.id"
                  class="h-1.5 w-1.5 bg-blue-600 rounded-full"
                  aria-hidden="true"
                ></span>
                <span
                  v-else
                  class="h-1.5 w-1.5 bg-gray-300 rounded-full"
                  aria-hidden="true"
                ></span>
              </template>
              <template v-else-if="stage.status == 'RUNNING'">
                <span
                  class="h-2.5 w-2.5 bg-blue-600 rounded-full"
                  style="
                    animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
                  "
                  aria-hidden="true"
                ></span>
              </template>
              <template v-else-if="stage.status == 'DONE'">
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
              <template v-else-if="stage.status == 'FAILED'">
                <span
                  class="h-2.5 w-2.5 rounded-full text-center pb-6 font-medium text-base"
                  aria-hidden="true"
                  >!</span
                >
              </template>
              <template v-else-if="stage.status == 'SKIPPED'">
                <svg
                  class="w-4 h-4"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    fill-rule="evenodd"
                    d="M10.293 15.707a1 1 0 010-1.414L14.586 10l-4.293-4.293a1 1 0 111.414-1.414l5 5a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0z"
                    clip-rule="evenodd"
                  ></path>
                  <path
                    fill-rule="evenodd"
                    d="M4.293 15.707a1 1 0 010-1.414L8.586 10 4.293 5.707a1 1 0 011.414-1.414l5 5a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0z"
                    clip-rule="evenodd"
                  ></path>
                </svg>
              </template>
            </div>
            <span class="ml-4 text-sm" :class="stageTextClass(stage)">{{
              stage.title
            }}</span>
          </span>
        </div>

        <!-- Arrow separator -->
        <div
          v-if="index != stageList.length - 1"
          class="hidden md:block absolute top-0 right-0 h-full w-5"
          aria-hidden="true"
        >
          <svg
            class="h-full w-full text-gray-300"
            viewBox="0 0 22 80"
            fill="none"
            preserveAspectRatio="none"
          >
            <path
              d="M0 -2L20 40L0 82"
              vector-effect="non-scaling-stroke"
              stroke="currentcolor"
              stroke-linejoin="round"
            />
          </svg>
        </div>
      </li>
    </ol>
  </nav>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { Task, StageId } from "../types";
import { activeStage } from "../utils";

interface FlowItem {
  id: StageId;
  title: string;
  status: string;
  link: () => string;
}

export default {
  name: "TaskFlow",
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  components: {},
  setup(props, ctx) {
    const stageList = computed<FlowItem[]>(() => {
      return props.task.stageProgressList.map((stageProgress) => {
        return {
          id: stageProgress.id,
          title: stageProgress.name,
          status: stageProgress.status,
          link: (): string => {
            return `/task/${props.task.id}`;
          },
        };
      });
    });

    const stageIconClass = (stage: FlowItem) => {
      switch (stage.status) {
        case "PENDING":
          if (activeStage(props.task).id === stage.id) {
            return "bg-white border-2 border-blue-600 text-blue-600 ";
          }
          return "bg-white border-2 border-gray-300";
        case "RUNNING":
          return "bg-white border-2 border-blue-600 text-blue-600";
        case "DONE":
          return "bg-success text-white";
        case "FAILED":
          return "bg-error text-white";
        case "SKIPPED":
          return "bg-white border-2 text-gray-400 border-gray-400";
      }
    };

    const stageTextClass = (stage: FlowItem) => {
      let textClass =
        activeStage(props.task).id === stage.id
          ? "font-medium "
          : "font-normal ";
      switch (stage.status) {
        case "SKIPPED":
          return textClass + "text-gray-500";
        case "DONE":
          return textClass + "text-control";
        case "PENDING":
          if (activeStage(props.task).id === stage.id) {
            return textClass + "text-blue-600";
          }
          return textClass + "text-control";
        case "RUNNING":
          return textClass + "text-blue-600";
        case "FAILED":
          return textClass + "text-red-500";
      }
    };

    return {
      stageList,
      activeStage,
      stageIconClass,
      stageTextClass,
    };
  },
};
</script>
