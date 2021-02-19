<template>
  <div
    id="task-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <!-- Highlight Panel -->
    <div class="bg-white px-4 py-6 lg:border-t lg:border-block-border">
      <TaskHighlightPanel :task="state.task">
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
    <TaskFlow
      v-if="!state.new && state.task.attributes.stageProgressList.length > 1"
      :task="state.task"
    />

    <!-- Output Panel -->
    <TaskOutputPanel
      v-if="!state.new && outputFieldList.length > 0"
      :task="state.task"
      :fieldList="outputFieldList"
      @update-custom-field="updateCustomField"
    />

    <!-- Main Content -->
    <main
      class="flex-1 relative overflow-y-auto focus:outline-none"
      tabindex="-1"
    >
      <div class="py-6">
        <div class="flex max-w-3xl mx-auto px-4 sm:px-6 lg:max-w-full">
          <div
            class="flex flex-col flex-1 min-w-0 lg:col-span-2 lg:pr-8 lg:border-r lg:border-gray-200"
          >
            <div>
              <TaskContentBar v-if="false" :task="state.task" />
              <TaskSidebar
                class="lg:hidden"
                :task="state.task"
                :fieldList="inputFieldList"
                @update-task-status="updateTaskStatus"
                @update-custom-field="updateCustomField"
              />
              <div class="lg:hidden my-4 border-t border-block-border" />
              <TaskContent :task="state.task" />
            </div>
            <section aria-labelledby="activity-title" class="mt-8 lg:mt-10">
              <TaskActivityPanel :task="state.task" />
            </section>
          </div>
          <TaskSidebar
            class="hidden lg:block lg:w-64 lg:pl-8 xl:w-72"
            :task="state.task"
            :fieldList="inputFieldList"
            @update-task-status="updateTaskStatus"
            @update-custom-field="updateCustomField"
          />
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
import { humanize, idFromSlug, taskSlug, activeStage } from "../utils";
import TaskHighlightPanel from "../views/TaskHighlightPanel.vue";
import TaskFlow from "../views/TaskFlow.vue";
import TaskOutputPanel from "../views/TaskOutputPanel.vue";
import TaskContentBar from "../views/TaskContentBar.vue";
import TaskContent from "../views/TaskContent.vue";
import TaskActivityPanel from "../views/TaskActivityPanel.vue";
import TaskSidebar from "../views/TaskSidebar.vue";
import {
  User,
  Task,
  TaskNew,
  TaskPatch,
  TaskStatus,
  StageStatus,
} from "../types";
import { taskTemplateList, TaskField, TaskTemplate } from "../plugins";

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
  requiredRunnable: boolean;
  to: StageStatus;
}

const STAGE_TRANSITION_LIST: Transition[] = [
  {
    type: "RUN",
    actionName: "Run",
    actionType: ActionType.PRIMARY,
    requiredRunnable: true,
    to: "RUNNING",
  },
  {
    type: "RETRY",
    actionName: "Rerun",
    actionType: ActionType.PRIMARY,
    requiredRunnable: true,
    to: "RUNNING",
  },
  {
    type: "CANCEL",
    actionName: "Cancel",
    actionType: ActionType.PRIMARY,
    requiredRunnable: true,
    to: "PENDING",
  },
  {
    type: "SKIP",
    actionName: "Skip",
    actionType: ActionType.NORMAL,
    requiredRunnable: false,
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
    TaskContentBar,
    TaskContent,
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
      router.currentRoute.value.query.template || "bytebase.general";
    const newTaskTemplate = taskTemplateList.find(
      (template) => template.type === newTaskemplateName
    )!;
    if (!newTaskTemplate) {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: `Unknown template '${newTaskTemplate}'.`,
      });
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

    const taskTemplate = taskTemplateList.find(
      (template) => template.type == state.task.attributes.type
    )!;

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

    const updateCustomField = (field: TaskField, value: string) => {
      if (field.preprocessor) {
        value = field.preprocessor(value);
      }
      state.task.attributes.payload[field.id] = value;
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

    const patchTask = (taskPatch: TaskPatch) => {
      store
        .dispatch("task/patchTask", {
          taskId: (state.task as Task).id,
          taskPatch,
        })
        .then((updatedTask) => {
          state.task = cloneDeep(updatedTask);
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
          (!transition.requiredRunnable || stage.runnable)
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

    return {
      state,
      modalState,
      humanize,
      updateTaskStatus,
      updateCustomField,
      doCreate,
      tryChangeStageStatus,
      doChangeStageStatus,
      enableHighlightButton,
      applicableStageTransitionList,
      actionButtonClass,
      currentUser,
      outputFieldList,
      inputFieldList,
    };
  },
};
</script>
