<template>
  <div
    id="task-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <!-- Highlight Panel -->
    <div class="bg-white px-4 py-6 lg:border-t lg:border-block-border">
      <TaskHighlightPanel :task="state.task" :new="state.new">
        <template v-if="state.new">
          <button
            type="button"
            class="btn-primary px-4 py-2"
            @click.prevent="doCreate"
            :disabled="!enableHighlightButton(0)"
          >
            Create
          </button>
        </template>
        <!-- Action Button List -->
        <template
          v-else
          v-for="(transition, index) in applicableStageTransitionList()"
          :key="index"
        >
          <button
            type="button"
            class="px-4 py-2"
            :class="actionButtonClass(transition.actionType)"
            @click.prevent="tryChangeStageStatus(transition)"
            :disabled="!enableHighlightButton(index)"
          >
            {{ transition.actionName }}
          </button>
        </template>
      </TaskHighlightPanel>
    </div>

    <!-- Flow Bar -->
    <TaskFlow v-if="showTaskFlowBar" :task="state.task" />

    <!-- Output Panel -->
    <!-- Only render the top border if TaskFlow is not displayed, otherwise it would overlap with the bottom border of the TaskFlow -->
    <div
      v-if="showTaskOutputPanel"
      class="px-2 md:flex md:flex-col"
      :class="showTaskFlowBar ? '' : 'lg:border-t py-4'"
    >
      <TaskOutputPanel
        :task="state.task"
        :fieldList="outputFieldList"
        @update-custom-field="updateCustomField"
      />
    </div>

    <!-- Main Content -->
    <main
      class="flex-1 relative overflow-y-auto focus:outline-none"
      :class="
        showTaskFlowBar && !showTaskOutputPanel
          ? ''
          : 'lg:border-t lg:border-block-border'
      "
      tabindex="-1"
    >
      <div class="flex max-w-3xl mx-auto px-4 sm:px-6 lg:max-w-full">
        <div class="flex flex-col flex-1 lg:flex-row-reverse lg:col-span-2">
          <div
            class="py-6 lg:px-4 lg:w-80 xl:w-96 lg:border-l lg:border-block-border"
          >
            <TaskSidebar
              :task="state.task"
              :new="state.new"
              :fieldList="inputFieldList"
              @update-task-status="updateTaskStatus"
              @update-assignee="updateAssignee"
              @update-custom-field="updateCustomField"
            />
          </div>
          <div class="lg:hidden my-4 border-t border-block-border" />
          <div class="w-full py-6 pr-4">
            <TaskDescription
              :task="state.task"
              :new="state.new"
              @update-description="updateDescription"
            />
            <section
              v-if="!state.new"
              aria-labelledby="activity-title"
              class="mt-4 lg:mt-6"
            >
              <TaskActivityPanel
                :task="state.task"
                :taskTemplate="taskTemplate"
              />
            </section>
          </div>
        </div>
      </div>
    </main>
  </div>
  <BBAlert
    v-if="modalState.show"
    :style="'INFO'"
    :okText="modalState.okText"
    :cancelText="'No'"
    :title="modalState.title"
    :payload="modalState.payload"
    @ok="
      (transition) => {
        modalState.show = false;
        doChangeStageStatus(transition);
      }
    "
    @cancel="modalState.show = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { computed, onMounted, watchEffect, reactive, inject } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEmpty from "lodash-es/isEmpty";
import { UserStateSymbol } from "../components/ProvideUser.vue";
import { idFromSlug, taskSlug, activeStage } from "../utils";
import TaskHighlightPanel from "../views/TaskHighlightPanel.vue";
import TaskFlow from "../views/TaskFlow.vue";
import TaskOutputPanel from "../views/TaskOutputPanel.vue";
import TaskDescription from "../views/TaskDescription.vue";
import TaskActivityPanel from "../views/TaskActivityPanel.vue";
import TaskSidebar from "../views/TaskSidebar.vue";
import {
  User,
  Task,
  TaskNew,
  TaskType,
  TaskPatch,
  TaskStatus,
  StageStatus,
  UserDisplay,
} from "../types";
import { templateForType, TaskField } from "../plugins";

type StageTransitionType = "RUN" | "RETRY" | "CANCEL" | "SKIP";

const CREATOR_APPLICABLE_STAGE_ACTION_LIST: Map<
  StageStatus,
  StageTransitionType[]
> = new Map([
  ["PENDING", []],
  ["RUNNING", []],
  ["DONE", []],
  ["FAILED", []],
  ["SKIPPED", []],
]);

const ASSIGNEE_APPLICABLE_STAGE_ACTION_LIST: Map<
  StageStatus,
  StageTransitionType[]
> = new Map([
  ["PENDING", ["RUN", "SKIP"]],
  ["RUNNING", ["CANCEL"]],
  ["DONE", []],
  ["FAILED", ["RETRY", "SKIP"]],
  ["SKIPPED", []],
]);

const GUEST_APPLICABLE_STAGE_ACTION_LIST: Map<
  StageStatus,
  StageTransitionType[]
> = new Map([
  ["PENDING", []],
  ["RUNNING", []],
  ["DONE", []],
  ["FAILED", []],
  ["SKIPPED", []],
]);

// Use enum so it's easier to do numeric comparison to sort the button.
enum ActionType {
  SUCCESS = 0,
  PRIMARY = 1,
  NORMAL = 2,
}

interface Transition {
  type: StageTransitionType;
  actionName: string;
  actionType: ActionType;
  requireRunnable: boolean;
  to: StageStatus;
}

const STAGE_TRANSITION_LIST: Transition[] = [
  {
    type: "RUN",
    actionName: "Run",
    actionType: ActionType.PRIMARY,
    requireRunnable: true,
    to: "RUNNING",
  },
  {
    type: "RETRY",
    actionName: "Rerun",
    actionType: ActionType.PRIMARY,
    requireRunnable: true,
    to: "RUNNING",
  },
  {
    type: "CANCEL",
    actionName: "Cancel",
    actionType: ActionType.PRIMARY,
    requireRunnable: true,
    to: "PENDING",
  },
  {
    type: "SKIP",
    actionName: "Skip",
    actionType: ActionType.NORMAL,
    requireRunnable: false,
    to: "SKIPPED",
  },
];

interface LocalState {
  new: boolean;
  task: Task | TaskNew;
}

interface ModalState {
  show: boolean;
  okText: string;
  title: string;
  payload?: Transition;
}

export default {
  name: "TaskDetail",
  props: {
    taskSlug: {
      required: true,
      type: String,
    },
  },
  components: {
    TaskHighlightPanel,
    TaskFlow,
    TaskOutputPanel,
    TaskDescription,
    TaskActivityPanel,
    TaskSidebar,
  },

  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const isNew = computed(() => {
      return props.taskSlug.toLowerCase() == "new";
    });

    const newTaskemplateName =
      (router.currentRoute.value.query.template as TaskType) ||
      "bytebase.general";
    const newTaskTemplate = templateForType(newTaskemplateName);
    if (!newTaskTemplate) {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: `Unknown template '${newTaskTemplate}'.`,
      });
      return;
    }

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]();
    });

    const currentUser = inject<User>(UserStateSymbol);

    const refreshState = () => {
      return {
        new: isNew.value,
        task: isNew.value
          ? newTaskTemplate.buildTask({
              environmentList: environmentList.value,
              currentUser: currentUser!,
            })
          : cloneDeep(
              store.getters["task/taskById"](idFromSlug(props.taskSlug))
            ),
      };
    };

    const state = reactive<LocalState>(refreshState());
    const modalState = reactive<ModalState>({
      show: false,
      okText: "OK",
      title: "",
    });

    const taskTemplate = templateForType(state.task.attributes.type)!;

    const outputFieldList =
      taskTemplate.fieldList?.filter((item) => item.category == "OUTPUT") || [];
    const inputFieldList =
      taskTemplate.fieldList?.filter((item) => item.category == "INPUT") || [];

    const refreshTask = () => {
      const updatedState = refreshState();
      state.new = updatedState.new;
      state.task = updatedState.task;
    };

    watchEffect(refreshTask);

    onMounted(() => {
      // Always scroll to top, the scrollBehavior doesn't seem to work.
      // The hypothesis is that because the scroll bar is in the nested
      // route, thus setting the scrollBehavior in the global router
      // won't work.
      document.getElementById("task-detail-top")!.scrollIntoView();
    });

    const updateDescription = (
      newDescription: string,
      postUpdated: (updatedTask: Task) => void
    ) => {
      patchTask(
        {
          description: newDescription,
        },
        postUpdated
      );
    };

    const updateTaskStatus = (newStatus: TaskStatus) => {
      // if (newStatus === "DONE") {
      //   if (template.fieldList) {
      //     for (const field of template.fieldList.filter(
      //       (item) => item.category == "OUTPUT"
      //     )) {
      //       if (
      //         field.required &&
      //         isEmpty(state.task.attributes.payload[field.id])
      //       ) {
      //         return;
      //       }
      //     }
      //   }
      // }
      patchTask({
        status: newStatus,
      });
    };

    const updateAssignee = (newAssignee: UserDisplay) => {
      // if (newStatus === "DONE") {
      //   if (template.fieldList) {
      //     for (const field of template.fieldList.filter(
      //       (item) => item.category == "OUTPUT"
      //     )) {
      //       if (
      //         field.required &&
      //         isEmpty(state.task.attributes.payload[field.id])
      //       ) {
      //         return;
      //       }
      //     }
      //   }
      // }
      patchTask({
        assignee: newAssignee,
      });
    };

    const updateCustomField = (field: TaskField, value: string) => {
      if (field.preprocessor) {
        value = field.preprocessor(value);
      }
      state.task.attributes.payload[field.id] = value;
      if (!state.new) {
        patchTask({
          payload: state.task.attributes.payload,
        });
      }
    };

    const doCreate = () => {
      store
        .dispatch("task/createTask", state.task)
        .then((createdTask) => {
          router.push(
            `/task/${taskSlug(createdTask.attributes.name, createdTask.id)}`
          );
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const patchTask = (
      taskPatch: TaskPatch,
      postUpdated?: (updatedTask: Task) => void
    ) => {
      store
        .dispatch("task/patchTask", {
          taskId: (state.task as Task).id,
          taskPatch: {
            ...taskPatch,
            producer: {
              id: currentUser!.id,
              name: currentUser!.attributes.name,
            },
          },
        })
        .then((updatedTask) => {
          if (postUpdated) {
            postUpdated(updatedTask);
          }
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const tryChangeStageStatus = (transition: Transition) => {
      modalState.okText = transition.actionName;
      modalState.title =
        transition.actionName +
        ' "' +
        activeStage(state.task as Task).name +
        '" ?';
      modalState.payload = transition;
      modalState.show = true;
    };

    const doChangeStageStatus = (transition: Transition) => {
      patchTask({
        stageProgressList: [
          {
            id: activeStage(state.task as Task).id,
            status: transition.to,
          },
        ],
      });
    };

    const enableHighlightButton = (buttonIndex: number): boolean => {
      if (state.new) {
        // Create
        if (buttonIndex == 0) {
          if (taskTemplate.fieldList) {
            for (const field of taskTemplate.fieldList.filter(
              (item) => item.category == "INPUT"
            )) {
              if (
                field.required &&
                isEmpty(state.task.attributes.payload[field.id])
              ) {
                return false;
              }
            }
          }
        }
      }
      return true;
    };

    const applicableStageTransitionList = () => {
      return STAGE_TRANSITION_LIST.filter((transition) => {
        const actionListForRole =
          currentUser!.id === state.task.attributes.creator.id
            ? CREATOR_APPLICABLE_STAGE_ACTION_LIST
            : currentUser!.id === state.task.attributes.assignee?.id
            ? ASSIGNEE_APPLICABLE_STAGE_ACTION_LIST
            : GUEST_APPLICABLE_STAGE_ACTION_LIST;
        const stage = activeStage(state.task as Task);
        return (
          stage.type === "ENVIRONMENT" &&
          actionListForRole.get(stage.status)!.includes(transition.type) &&
          (!transition.requireRunnable || stage.runnable)
        );
      }).sort((a, b) => b.actionType - a.actionType);
    };

    const actionButtonClass = (actionType: ActionType) => {
      switch (actionType) {
        case ActionType.SUCCESS:
          return "btn-success";
        case ActionType.PRIMARY:
          return "btn-primary";
        case ActionType.NORMAL:
          return "btn-normal";
      }
    };

    const showTaskFlowBar = computed(() => {
      return !state.new && state.task.attributes.stageProgressList.length > 1;
    });

    const showTaskOutputPanel = computed(() => {
      return false && !state.new && outputFieldList.length > 0;
    });

    return {
      state,
      modalState,
      updateDescription,
      updateTaskStatus,
      updateAssignee,
      updateCustomField,
      doCreate,
      tryChangeStageStatus,
      doChangeStageStatus,
      enableHighlightButton,
      applicableStageTransitionList,
      actionButtonClass,
      currentUser,
      taskTemplate,
      outputFieldList,
      inputFieldList,
      showTaskFlowBar,
      showTaskOutputPanel,
    };
  },
};
</script>
