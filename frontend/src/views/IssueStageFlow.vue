<template>
  <nav aria-label="Progress">
    <ol
      class="border-t border-b border-block-border divide-y divide-gray-300 lg:flex lg:divide-y-0"
    >
      <li
        v-for="(stage, index) in stageList"
        :key="index"
        class="relative md:flex-1 md:flex"
      >
        <div
          class="cursor-default group flex items-center justify-between w-full"
        >
          <span class="pl-4 py-3 flex items-center text-sm font-medium">
            <div
              class="relative w-6 h-6 flex items-center justify-center rounded-full select-none"
              :class="stageIconClass(stage)"
            >
              <template v-if="stage.status === 'PENDING'">
                <span
                  v-if="activeStage(issue).id === stage.id"
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
            <span class="ml-2 text-sm" :class="stageTextClass(stage)">{{
              stage.title
            }}</span>
          </span>
          <div
            v-if="
              activeStage(issue).id === stage.id &&
              applicableStageTransitionList.length > 0
            "
            class="flex flex-row space-x-1 mr-4"
          >
            <button
              class="btn-icon"
              @click.prevent="
                tryChangeStageStatus(stage.id, applicableStageTransitionList[0])
              "
            >
              <template v-if="applicableStageTransitionList[0].type == 'RUN'">
                <svg
                  class="w-6 h-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
                  ></path>
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  ></path>
                </svg>
              </template>
              <template
                v-else-if="applicableStageTransitionList[0].type == 'RETRY'"
              >
                <svg
                  class="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                  ></path>
                </svg>
              </template>
              <template
                v-else-if="applicableStageTransitionList[0].type == 'STOP'"
              >
                <svg
                  class="w-6 h-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  ></path>
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"
                  ></path>
                </svg>
              </template>
            </button>
            <template v-if="applicableStageTransitionList.length > 1">
              <button
                class="btn-icon"
                @click.prevent="$refs.menu.toggle($event)"
              >
                <svg
                  class="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
                  ></path>
                </svg>
              </button>
              <BBContextMenu
                ref="menu"
                class="origin-bottom-right absolute mt-6 -right-8 w-16 rounded-md shadow-lg"
              >
                <template
                  v-for="(
                    transition, index
                  ) in applicableStageTransitionList.slice(1)"
                  :key="index"
                >
                  <a
                    @click.prevent="tryChangeStageStatus(stage.id, transition)"
                    class="menu-item"
                    role="menuitem"
                  >
                    {{ transition.actionName }}
                  </a>
                </template>
              </BBContextMenu>
            </template>
          </div>
        </div>

        <!-- Arrow separator -->
        <div
          v-if="index != stageList.length - 1"
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
  <BBModal
    v-if="modalState.show"
    :title="modalState.title"
    @close="modalState.show = false"
  >
    <StageStatusTransitionForm
      :okText="modalState.okText"
      :issue="issue"
      :transition="modalState.transition"
      @submit="
        (transition, comment) => {
          modalState.show = false;
          doChangeStageStatus(modalState.stageId, transition, comment);
        }
      "
      @cancel="
        () => {
          modalState.show = false;
        }
      "
    />
  </BBModal>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";
import StageStatusTransitionForm from "../components/StageStatusTransitionForm.vue";
import {
  Issue,
  StageId,
  StageStatus,
  StageStatusTransitionType,
  StageStatusTransition,
  STAGE_TRANSITION_LIST,
  UNKNOWN_ID,
} from "../types";
import { activeStage } from "../utils";

// The first transition in the list is the primary action and the rests are
// the normal action which is hidden in the vertical dots icon.
const CREATOR_APPLICABLE_STAGE_ACTION_LIST: Map<
  StageStatus,
  StageStatusTransitionType[]
> = new Map([
  ["PENDING", []],
  ["RUNNING", []],
  ["DONE", []],
  ["FAILED", []],
  ["SKIPPED", []],
]);

const ASSIGNEE_APPLICABLE_STAGE_ACTION_LIST: Map<
  StageStatus,
  StageStatusTransitionType[]
> = new Map([
  ["PENDING", ["RUN", "SKIP"]],
  ["RUNNING", ["STOP"]],
  ["DONE", []],
  ["FAILED", ["RETRY", "SKIP"]],
  ["SKIPPED", []],
]);

interface FlowItem {
  id: StageId;
  title: string;
  status: string;
  link: () => string;
}

interface ModalState {
  stageId: StageId;
  show: boolean;
  okText: string;
  title: string;
  transition?: StageStatusTransition;
}

export default {
  name: "IssueStageFlow",
  emits: ["change-stage-status"],
  props: {
    issue: {
      required: true,
      type: Object as PropType<Issue>,
    },
  },
  components: { StageStatusTransitionForm },
  setup(props, { emit }) {
    const store = useStore();

    const modalState = reactive<ModalState>({
      stageId: UNKNOWN_ID,
      show: false,
      okText: "OK",
      title: "",
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const stageList = computed<FlowItem[]>(() => {
      return props.issue.stageList.map((stage) => {
        return {
          id: stage.id,
          title: stage.name,
          status: stage.status,
          link: (): string => {
            return `/issue/${props.issue.id}`;
          },
        };
      });
    });

    const stageIconClass = (stage: FlowItem) => {
      switch (stage.status) {
        case "PENDING":
          if (activeStage(props.issue).id === stage.id) {
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
        activeStage(props.issue).id === stage.id
          ? "font-medium "
          : "font-normal ";
      switch (stage.status) {
        case "SKIPPED":
          return textClass + "text-gray-500";
        case "DONE":
          return textClass + "text-control";
        case "PENDING":
          if (activeStage(props.issue).id === stage.id) {
            return textClass + "text-blue-600";
          }
          return textClass + "text-control";
        case "RUNNING":
          return textClass + "text-blue-600";
        case "FAILED":
          return textClass + "text-red-500";
      }
    };

    const applicableStageTransitionList = computed(() => {
      const stage = activeStage(props.issue as Issue);
      const list: StageStatusTransitionType[] = [];
      if (currentUser.value.id === (props.issue as Issue).assignee?.id) {
        list.push(...ASSIGNEE_APPLICABLE_STAGE_ACTION_LIST.get(stage.status)!);
      }
      if (currentUser.value.id === (props.issue as Issue).creator.id) {
        CREATOR_APPLICABLE_STAGE_ACTION_LIST.get(stage.status)!.forEach(
          (item) => {
            if (list.indexOf(item) == -1) {
              list.push(item);
            }
          }
        );
      }
      return list
        .filter((type) => {
          const transition = STAGE_TRANSITION_LIST.get(type)!;
          return !transition.requireRunnable || stage.runnable;
        })
        .map((type) => STAGE_TRANSITION_LIST.get(type));
    });

    const tryChangeStageStatus = (
      stageId: StageId,
      transition: StageStatusTransition
    ) => {
      modalState.stageId = stageId;
      modalState.okText = transition.actionName;
      modalState.title =
        transition.actionName +
        ' "' +
        activeStage(props.issue as Issue).name +
        '" ?';
      modalState.transition = transition;
      modalState.show = true;
    };

    const doChangeStageStatus = (
      stageId: StageId,
      transition: StageStatusTransition,
      comment?: string
    ) => {
      emit("change-stage-status", stageId, transition.to, comment);
    };

    return {
      modalState,
      stageList,
      activeStage,
      stageIconClass,
      stageTextClass,
      applicableStageTransitionList,
      tryChangeStageStatus,
      doChangeStageStatus,
    };
  },
};
</script>
