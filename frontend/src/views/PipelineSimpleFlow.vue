<template>
  <nav aria-label="Pipeline">
    <ol
      class="
        border-t border-b border-block-border
        divide-y divide-gray-300
        lg:flex lg:divide-y-0
      "
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
              class="
                relative
                w-6
                h-6
                flex flex-shrink-0
                items-center
                justify-center
                rounded-full
                select-none
              "
              :class="flowItemIconClass(item)"
            >
              <template v-if="item.taskStatus === 'PENDING'">
                <span
                  v-if="activeTask(pipeline).id === item.taskID"
                  class="h-2 w-2 bg-info rounded-full"
                  aria-hidden="true"
                ></span>
                <span
                  v-else
                  class="h-1.5 w-1.5 bg-control rounded-full"
                  aria-hidden="true"
                ></span>
              </template>
              <template v-else-if="item.taskStatus === 'PENDING_APPROVAL'">
                <svg
                  class="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
                  ></path>
                </svg>
              </template>
              <template v-else-if="item.taskStatus == 'RUNNING'">
                <span
                  class="h-2.5 w-2.5 bg-info rounded-full"
                  style="
                    animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
                  "
                  aria-hidden="true"
                ></span>
              </template>
              <template v-else-if="item.taskStatus == 'DONE'">
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
              <template v-else-if="item.taskStatus == 'FAILED'">
                <span
                  class="
                    h-2.5
                    w-2.5
                    rounded-full
                    text-center
                    pb-6
                    font-medium
                    text-base
                  "
                  aria-hidden="true"
                  >!</span
                >
              </template>
            </div>
            <div
              class="
                hidden
                cursor-pointer
                hover:underline
                lg:ml-4 lg:flex lg:flex-col
              "
              :class="flowItemTextClass(item)"
              @click.prevent="clickItem(item)"
            >
              <span class="text-xs">
                {{ item.stageName }}
              </span>
              <span class="text-sm">{{ item.taskName }}</span>
            </div>
            <div
              class="ml-4 group cursor-pointer grid grid-cols-2 lg:hidden"
              :class="flowItemTextClass(item)"
              @click.prevent="clickItem(item)"
            >
              <span class="col-span-1 text-sm w-32">{{ item.stageName }} </span>
              <span class="col-span-1 text-sm">{{ item.taskName }}</span>
            </div>
            <div class="tooltip-wrapper" @click.prevent="clickItem(item)">
              <span class="tooltip">Missing SQL statement</span>
              <span
                v-if="!item.valid"
                class="
                  ml-2
                  w-5
                  h-5
                  flex
                  justify-center
                  rounded-full
                  select-none
                  bg-error
                  text-white
                  hover:bg-error-hover
                "
              >
                <span class="text-center font-normal" aria-hidden="true"
                  >!</span
                >
              </span>
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
import { useRouter } from "vue-router";
import {
  Pipeline,
  TaskStatus,
  TaskID,
  Stage,
  StageID,
  StageCreate,
  PipelineCreate,
  Task,
  TaskCreate,
} from "../types";
import { activeTask, activeTaskInStage } from "../utils";
import isEmpty from "lodash-es/isEmpty";

interface FlowItem {
  stageID: StageID;
  stageName: string;
  taskID: TaskID;
  taskName: string;
  taskStatus: TaskStatus;
  valid: boolean;
}

export default {
  name: "PipelineSimpleFlow",
  emits: ["select-stage-id"],
  props: {
    create: {
      required: true,
      type: Boolean,
    },
    pipeline: {
      required: true,
      type: Object as PropType<Pipeline | PipelineCreate>,
    },
    selectedStage: {
      required: true,
      type: Object as PropType<Stage | StageCreate>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const router = useRouter();

    const itemList = computed<FlowItem[]>(() => {
      return props.pipeline.stageList.map((stage, index) => {
        let activeTask = stage.taskList[0];
        let taskName = activeTask.name;
        let valid = true;
        if (props.create) {
          for (const task of stage.taskList) {
            if (
              task.type == "bb.task.database.create" ||
              task.type == "bb.task.database.schema.update"
            ) {
              if (isEmpty((task as TaskCreate).statement)) {
                valid = false;
                break;
              }
            }
          }
        } else {
          activeTask = activeTaskInStage(stage as Stage);
          if (stage.taskList.length > 1) {
            for (let i = 0; i < stage.taskList.length; i++) {
              if ((stage.taskList[i] as Task).id == (activeTask as Task).id) {
                taskName = `${activeTask.name} (${i + 1}/${
                  stage.taskList.length
                })`;
                break;
              }
            }
          }
        }

        return {
          stageID: props.create ? index : (stage as Stage).id,
          stageName: stage.name,
          taskID: props.create ? index : (activeTask as Task).id,
          taskName: taskName,
          taskStatus: activeTask.status,
          valid,
        };
      });
    });

    const flowItemIconClass = (item: FlowItem) => {
      switch (item.taskStatus) {
        case "PENDING":
          if (
            !props.create &&
            activeTask(props.pipeline as Pipeline).id === item.taskID
          ) {
            return "bg-white border-2 border-info text-info ";
          }
          return "bg-white border-2 border-control";
        case "PENDING_APPROVAL":
          if (
            !props.create &&
            activeTask(props.pipeline as Pipeline).id === item.taskID
          ) {
            return "bg-white border-2 border-info text-info";
          }
          return "bg-white border-2 border-control";
        case "RUNNING":
          return "bg-white border-2 border-info text-info";
        case "DONE":
          return "bg-success text-white";
        case "FAILED":
          return "bg-error text-white";
      }
    };

    const flowItemTextClass = (item: FlowItem) => {
      let textClass =
        !props.create &&
        activeTask(props.pipeline as Pipeline).id === item.taskID
          ? "font-bold "
          : "font-normal ";
      // For create, since we don't have stage id yet, we just compare name instead.
      // Not 100% accurate, but should be fine most of the time.
      if (
        (props.create &&
          item.stageName == (props.selectedStage as StageCreate).name) ||
        (!props.create && item.stageID == (props.selectedStage as Stage).id)
      ) {
        textClass += "underline ";
      }
      switch (item.taskStatus) {
        case "DONE":
          return textClass + "text-control";
        case "PENDING":
        case "PENDING_APPROVAL":
          if (
            !props.create &&
            activeTask(props.pipeline as Pipeline).id === item.taskID
          ) {
            return textClass + "text-info";
          }
          return textClass + "text-control";
        case "RUNNING":
          return textClass + "text-info";
        case "FAILED":
          return textClass + "text-red-500";
      }
    };

    const clickItem = (item: FlowItem) => {
      emit("select-stage-id", item.stageID);
    };

    return {
      itemList,
      activeTask,
      flowItemIconClass,
      flowItemTextClass,
      clickItem,
    };
  },
};
</script>
