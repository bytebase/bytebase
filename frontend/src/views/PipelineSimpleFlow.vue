<template>
  <nav aria-label="Pipeline">
    <ol
      class="border-t border-b border-block-border divide-y divide-gray-300 lg:flex lg:divide-y-0"
    >
      <li
        v-for="(item, index) in itemList"
        :key="index"
        class="relative md:flex-1 md:flex"
      >
        <div
          class="cursor-default group flex items-center justify-between w-full"
        >
          <span class="pl-4 py-2 flex items-center text-sm font-medium">
            <div
              class="relative w-6 h-6 flex flex-shrink-0 items-center justify-center rounded-full select-none"
              :class="flowItemIconClass(item)"
            >
              <template v-if="item.stepStatus === 'PENDING'">
                <span
                  v-if="activeStep(pipeline).id === item.stepId"
                  class="h-1.5 w-1.5 bg-blue-600 rounded-full"
                  aria-hidden="true"
                ></span>
                <span
                  v-else
                  class="h-1.5 w-1.5 bg-gray-300 rounded-full"
                  aria-hidden="true"
                ></span>
              </template>
              <template v-else-if="item.stepStatus == 'RUNNING'">
                <span
                  class="h-2.5 w-2.5 bg-blue-600 rounded-full"
                  style="
                    animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
                  "
                  aria-hidden="true"
                ></span>
              </template>
              <template v-else-if="item.stepStatus == 'DONE'">
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
              <template v-else-if="item.stepStatus == 'FAILED'">
                <span
                  class="h-2.5 w-2.5 rounded-full text-center pb-6 font-medium text-base"
                  aria-hidden="true"
                  >!</span
                >
              </template>
              <template v-else-if="item.stepStatus == 'SKIPPED'">
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
            </div>
            <div class="hidden lg:ml-4 lg:flex lg:flex-col">
              <span class="text-xs" :class="flowItemTextClass(item)">{{
                item.taskName
              }}</span>
              <span class="text-sm" :class="flowItemTextClass(item)">{{
                item.stepName
              }}</span>
            </div>
            <div class="ml-4 grid grid-cols-2 lg:hidden">
              <span
                class="col-span-1 flex items-center text-sm w-32"
                :class="flowItemTextClass(item)"
                >{{ item.taskName }}</span
              >
              <span
                class="col-span-1 text-sm"
                :class="flowItemTextClass(item)"
                >{{ item.stepName }}</span
              >
            </div>
          </span>
        </div>

        <!-- Arrow separator -->
        <div
          v-if="index != itemList.length - 1"
          class="hidden lg:block absolute top-0 right-0 h-full w-5"
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
import { Pipeline, StepStatus, StepId } from "../types";
import { activeStep } from "../utils";

interface FlowItem {
  taskName: string;
  stepId: StepId;
  stepName: string;
  stepStatus: StepStatus;
}

export default {
  name: "PipelineSimpleFlow",
  emits: [],
  props: {
    pipeline: {
      required: true,
      type: Object as PropType<Pipeline>,
    },
  },
  components: {},
  setup(props, {}) {
    const itemList = computed<FlowItem[]>(() => {
      return props.pipeline.taskList.map((task) => {
        const step = task.stepList[0];
        return {
          taskName: task.name,
          stepId: step.id,
          stepName: step.name,
          stepStatus: step.status,
        };
      });
    });

    const flowItemIconClass = (item: FlowItem) => {
      switch (item.stepStatus) {
        case "PENDING":
          if (activeStep(props.pipeline).id === item.stepId) {
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

    const flowItemTextClass = (item: FlowItem) => {
      let textClass =
        activeStep(props.pipeline).id === item.stepId
          ? "font-medium "
          : "font-normal ";
      switch (item.stepStatus) {
        case "SKIPPED":
          return textClass + "text-gray-500";
        case "DONE":
          return textClass + "text-control";
        case "PENDING":
          if (activeStep(props.pipeline).id === item.stepId) {
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
      itemList,
      activeStep,
      flowItemIconClass,
      flowItemTextClass,
    };
  },
};
</script>
