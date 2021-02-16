<template>
  <div
    id="task-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <!-- Highlight Panel -->
    <div class="bg-white px-6 py-6 lg:border-t lg:border-block-border">
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
        <!-- Action Buttons only applicable to creator -->
        <template
          v-else
          v-for="(transition, index) in applicableTransitionList()"
          :key="index"
        >
          <button
            type="button"
            class="px-4 py-2"
            :class="actionButtonClass(transition.actionType)"
            @click.prevent="doChangeStatus(transition)"
            :disabled="!enableHighlightButton(index)"
          >
            {{ transition.actionName }}
          </button>
        </template>
      </TaskHighlightPanel>
    </div>

    <!-- Flow Bar -->
    <TaskFlow
      v-if="state.task.stageProgressList?.length > 0"
      :task="state.task"
    />

    <!-- Output Panel -->
    <TaskOutputPanel
      v-if="!state.new && template.outputFieldList?.length > 0"
      :task="state.task"
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
                @update-field="updateField"
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
            @update-field="updateField"
          />
        </div>
      </div>
    </main>
  </div>
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
import { taskTemplateList, TaskField } from "../plugins";

type TaskTransitionType =
  | "bytebase.task.resolve"
  | "bytebase.task.abort"
  | "bytebase.task.reopen";

type StageTransitionType =
  | "bytebase.stage.retry"
  | "bytebase.stage.cancel"
  | "bytebase.stage.skip";

type TransitionType = TaskTransitionType | StageTransitionType;

// Use enum so it's easier to do numeric comparison to sort the button.
enum ActionType {
  SUCCESS = 0,
  PRIMARY = 1,
  NORMAL = 2,
}

type RoleType = "ASSIGNEE" | "CREATOR" | "GUEST";

interface SourceWorkflowStatus {
  taskStatus: TaskStatus[];
  stageStatus: StageStatus[];
}

interface WorkflowStatus {
  taskStatus: TaskStatus;
  stageStatus: StageStatus;
}

interface Transition {
  type: TransitionType;
  actionName: string;
  actionType: ActionType;
  from: SourceWorkflowStatus;
  to: (status: WorkflowStatus) => WorkflowStatus;
  allowedRoleList: RoleType[];
}

const TRANSITION_LIST: Transition[] = [
  {
    type: "bytebase.task.resolve",
    actionName: "Resolve",
    actionType: ActionType.SUCCESS,
    from: {
      taskStatus: ["OPEN"],
      stageStatus: ["PENDING", "DONE", "SKIPPED"],
    },
    to: (from: WorkflowStatus) => {
      return {
        taskStatus: "DONE",
        stageStatus: from.stageStatus == "PENDING" ? "DONE" : from.stageStatus,
      };
    },
    allowedRoleList: ["ASSIGNEE", "CREATOR"],
  },
  {
    type: "bytebase.task.abort",
    actionName: "Abort",
    actionType: ActionType.NORMAL,
    from: {
      taskStatus: ["OPEN"],
      stageStatus: ["PENDING", "FAILED"],
    },
    to: (from: WorkflowStatus) => {
      return {
        taskStatus: "CANCELED",
        stageStatus: "SKIPPED",
      };
    },
    allowedRoleList: ["CREATOR"],
  },
  {
    type: "bytebase.task.reopen",
    actionName: "Reopen",
    actionType: ActionType.NORMAL,
    from: {
      taskStatus: ["DONE", "CANCELED"],
      stageStatus: ["PENDING", "FAILED"],
    },
    to: (from: WorkflowStatus) => {
      return {
        taskStatus: "OPEN",
        stageStatus: "PENDING",
      };
    },
    allowedRoleList: ["ASSIGNEE", "CREATOR"],
  },
];

interface LocalState {
  new: boolean;
  task: Task | TaskNew;
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

    const templateName =
      router.currentRoute.value.query.template || "bytebase.general";
    const taskTemplate = taskTemplateList.find(
      (template) => template.type === templateName
    )!;
    if (!taskTemplate) {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: `Unknown template '${templateName}'.`,
      });
    }

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]();
    });

    const currentUser = inject<User>(UserStateSymbol);

    const refreshState = () => {
      return {
        new: props.taskSlug.toLowerCase() == "new",
        task:
          props.taskSlug.toLowerCase() == "new"
            ? taskTemplate.buildTask({
                environmentList: environmentList.value,
                currentUser: currentUser!,
              })
            : cloneDeep(
                store.getters["task/taskById"](idFromSlug(props.taskSlug))
              ),
      };
    };

    const state = reactive<LocalState>(refreshState());

    const template = taskTemplateList.find(
      (template) => template.type == state.task.attributes.type
    )!;

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

    const updateField = (field: TaskField, value: string) => {
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

    const doChangeStatus = (transition: Transition) => {
      patchTask({
        status: transition.to({
          taskStatus: (state.task as Task).attributes.status,
          stageStatus: activeStage(state.task as Task).status,
        }).taskStatus,
      });
    };

    const enableHighlightButton = (buttonIndex: number): boolean => {
      if (state.new) {
        // Create
        if (buttonIndex == 0) {
          if (template.fieldList) {
            for (const field of template.fieldList) {
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

    const applicableTransitionList = () => {
      return TRANSITION_LIST.filter((transition) => {
        const role: RoleType =
          currentUser!.id === state.task.attributes.creator.id
            ? "CREATOR"
            : currentUser!.id === state.task.attributes.assignee?.id
            ? "ASSIGNEE"
            : "GUEST";
        return (
          transition.from.taskStatus.includes(
            (state.task as Task).attributes.status
          ) &&
          transition.from.stageStatus.includes(
            activeStage(state.task as Task).status
          ) &&
          transition.allowedRoleList.includes(role)
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
      template,
      humanize,
      updateField,
      doCreate,
      doChangeStatus,
      enableHighlightButton,
      applicableTransitionList,
      actionButtonClass,
      currentUser,
    };
  },
};
</script>
