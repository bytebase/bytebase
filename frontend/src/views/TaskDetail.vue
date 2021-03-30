<template>
  <div
    id="task-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <!-- Highlight Panel -->
    <div class="bg-white px-4 pb-4">
      <TaskHighlightPanel
        :task="state.task"
        :new="state.new"
        :allowEdit="allowEditFields"
        @update-name="updateName"
      >
        <template v-if="state.new">
          <button
            type="button"
            class="btn-primary px-4 py-2"
            @click.prevent="doCreate"
            :disabled="!allowCreate"
          >
            Create
          </button>
        </template>
        <!-- Action Button List -->
        <div
          v-else-if="applicableStatusTransitionList.length > 0"
          class="flex flex-row-reverse"
        >
          <template
            v-for="(transition, index) in applicableStatusTransitionList"
            :key="index"
          >
            <button
              type="button"
              :class="
                index == 0
                  ? transition.type == 'RESOLVE'
                    ? 'btn-success'
                    : 'btn-normal'
                  : 'btn-normal mr-2'
              "
              :disabled="!allowTransition(transition)"
              @click.prevent="
                tryStartTaskStatusTransition(transition.type, () => {})
              "
            >
              {{ transition.actionName }}
            </button>
          </template>
        </div>
      </TaskHighlightPanel>
    </div>

    <!-- Stage Flow Bar -->
    <TaskStageFlow
      v-if="showTaskStageFlowBar"
      :task="state.task"
      @change-stage-status="changeStageStatus"
    />

    <!-- Output Panel -->
    <!-- Only render the top border if TaskStageFlow is not displayed, otherwise it would overlap with the bottom border of the TaskStageFlow -->
    <div
      v-if="showTaskOutputPanel"
      class="px-2 py-4 md:flex md:flex-col"
      :class="showTaskStageFlowBar ? '' : 'lg:border-t'"
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
        showTaskStageFlowBar && !showTaskOutputPanel
          ? ''
          : 'lg:border-t lg:border-block-border'
      "
      tabindex="-1"
    >
      <div class="flex max-w-3xl mx-auto px-6 lg:max-w-full">
        <div class="flex flex-col flex-1 lg:flex-row-reverse lg:col-span-2">
          <div
            class="py-6 lg:pl-4 lg:w-96 xl:w-112 lg:border-l lg:border-block-border"
          >
            <TaskSidebar
              :task="state.task"
              :new="state.new"
              :fieldList="inputFieldList"
              :allowEdit="allowEditFields"
              @update-assignee-id="updateAssigneeId"
              @update-custom-field="updateCustomField"
            />
          </div>
          <div class="lg:hidden border-t border-block-border" />
          <div class="w-full py-6 pr-4">
            <section v-if="showTaskSqlPanel" class="border-b mb-4">
              <TaskSqlPanel
                :task="state.task"
                :new="state.new"
                :rollback="false"
                :allowEdit="allowEditFields"
                @update-sql="updateSql"
              />
            </section>
            <section v-if="showTaskRollbackSqlPanel" class="border-b mb-4">
              <TaskSqlPanel
                :task="state.task"
                :new="state.new"
                :rollback="true"
                :allowEdit="allowEditFields"
                @update-sql="updateRollbackSql"
              />
            </section>
            <TaskDescriptionPanel
              :task="state.task"
              :new="state.new"
              :allowEdit="allowEditFields"
              @update-description="updateDescription"
            />
            <section
              v-if="!state.new"
              aria-labelledby="activity-title"
              class="mt-4"
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
  <BBModal
    v-if="updateStatusModalState.show"
    :title="updateStatusModalState.title"
    @close="updateStatusModalState.show = false"
  >
    <TaskStatusTransitionForm
      :okText="updateStatusModalState.okText"
      :task="state.task"
      :transition="updateStatusModalState.payload.transition"
      :outputFieldList="outputFieldList"
      @submit="
        (outputValueList, comment) => {
          updateStatusModalState.show = false;
          doTaskStatusTransition(
            updateStatusModalState.payload,
            outputValueList,
            comment
          );
        }
      "
      @cancel="
        () => {
          updateStatusModalState.show = false;
        }
      "
    />
  </BBModal>
</template>

<script lang="ts">
import {
  computed,
  onMounted,
  watch,
  watchEffect,
  reactive,
  ref,
  ComputedRef,
} from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEmpty from "lodash-es/isEmpty";
import isEqual from "lodash-es/isEqual";
import { idFromSlug, taskSlug, pendingResolve } from "../utils";
import TaskHighlightPanel from "../views/TaskHighlightPanel.vue";
import TaskStageFlow from "./TaskStageFlow.vue";
import TaskOutputPanel from "../views/TaskOutputPanel.vue";
import TaskSqlPanel from "../views/TaskSqlPanel.vue";
import TaskDescriptionPanel from "./TaskDescriptionPanel.vue";
import TaskActivityPanel from "../views/TaskActivityPanel.vue";
import TaskSidebar from "../views/TaskSidebar.vue";
import TaskStatusTransitionForm from "../components/TaskStatusTransitionForm.vue";
import {
  Task,
  TaskNew,
  TaskType,
  TaskPatch,
  TaskStatus,
  TaskStatusTransition,
  TaskStatusTransitionType,
  StageProgressPatch,
  PrincipalId,
  TASK_STATUS_TRANSITION_LIST,
} from "../types";
import {
  defaulTemplate,
  templateForType,
  TaskField,
  TaskTemplate,
} from "../plugins";

// The first transition in the list is the primary action and the rests are
// the normal action. For now there are at most 1 primary 1 normal action.
const CREATOR_APPLICABLE_ACTION_LIST: Map<
  TaskStatus,
  TaskStatusTransitionType[]
> = new Map([
  ["OPEN", ["ABORT"]],
  ["DONE", ["REOPEN"]],
  ["CANCELED", ["REOPEN"]],
]);

const ASSIGNEE_APPLICABLE_ACTION_LIST: Map<
  TaskStatus,
  TaskStatusTransitionType[]
> = new Map([
  ["OPEN", ["RESOLVE", "ABORT"]],
  ["DONE", ["REOPEN"]],
  ["CANCELED", ["REOPEN"]],
]);

type UpdateStatusModalStatePayload = {
  transition: TaskStatusTransition;
  didTransit: () => {};
};

interface UpdateStatusModalState {
  show: boolean;
  style: string;
  okText: string;
  title: string;
  payload?: UpdateStatusModalStatePayload;
}

interface LocalState {
  new: boolean;
  task: ComputedRef<Task | TaskNew>;
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
    TaskStageFlow,
    TaskOutputPanel,
    TaskSqlPanel,
    TaskDescriptionPanel,
    TaskActivityPanel,
    TaskSidebar,
    TaskStatusTransitionForm,
  },

  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const updateStatusModalState = reactive<UpdateStatusModalState>({
      show: false,
      style: "INFO",
      okText: "OK",
      title: "",
    });

    onMounted(() => {
      // Always scroll to top, the scrollBehavior doesn't seem to work.
      // The hypothesis is that because the scroll bar is in the nested
      // route, thus setting the scrollBehavior in the global router
      // won't work.
      document.getElementById("task-detail-top")!.scrollIntoView();
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]();
    });

    const isNew = computed(() => {
      return props.taskSlug.toLowerCase() == "new";
    });

    let newTaskTemplate = ref<TaskTemplate>(defaulTemplate());

    const refreshTemplate = () => {
      const taskType = router.currentRoute.value.query.template as TaskType;
      if (taskType) {
        const template = templateForType(taskType);
        if (template) {
          newTaskTemplate.value = template;
        } else {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "CRITICAL",
            title: `Unknown template '${taskType}'.`,
            description: "Fallback to the default template",
          });
        }
      }

      if (!newTaskTemplate.value) {
        newTaskTemplate.value = defaulTemplate();
      }
    };

    // Vue doesn't natively react to query parameter change
    // so we need to manually watch here.
    watch(
      () => router.currentRoute.value.query.template,
      (curTemplate, prevTemplate) => {
        refreshTemplate();
      }
    );

    watchEffect(refreshTemplate);

    const refreshState = () => {
      const newTask: TaskNew = newTaskTemplate.value.buildTask({
        environmentList: environmentList.value,
        currentUser: currentUser.value,
      });

      newTask.creatorId = currentUser.value.id;

      if (router.currentRoute.value.query.name) {
        newTask.name = router.currentRoute.value.query.name as string;
      }
      if (router.currentRoute.value.query.description) {
        newTask.description = router.currentRoute.value.query
          .description as string;
      }
      if (router.currentRoute.value.query.sql) {
        newTask.sql = router.currentRoute.value.query.sql as string;
      }
      if (router.currentRoute.value.query.rollbacksql) {
        newTask.rollbackSql = router.currentRoute.value.query
          .rollbacksql as string;
      }
      if (router.currentRoute.value.query.assignee) {
        newTask.assigneeId = router.currentRoute.value.query
          .assignee as PrincipalId;
      }

      for (const field of newTaskTemplate.value.fieldList.filter(
        (item) => item.category == "INPUT"
      )) {
        const value = router.currentRoute.value.query[field.slug];
        if (value) {
          newTask.payload[field.id] = value;
        }
      }

      return {
        new: isNew.value,
        task: isNew.value
          ? newTask
          : cloneDeep(
              store.getters["task/taskById"](idFromSlug(props.taskSlug))
            ),
      };
    };

    const state = reactive<LocalState>(refreshState());

    const refreshTask = () => {
      const updatedState = refreshState();
      state.new = updatedState.new;
      state.task = updatedState.task;
    };

    watchEffect(refreshTask);

    const taskTemplate = computed(
      () => templateForType(state.task.type) || defaulTemplate()
    );

    const outputFieldList = computed(
      () =>
        taskTemplate.value.fieldList.filter(
          (item) => item.category == "OUTPUT"
        ) || []
    );
    const inputFieldList = computed(
      () =>
        taskTemplate.value.fieldList.filter(
          (item) => item.category == "INPUT"
        ) || []
    );

    const updateName = (
      newName: string,
      postUpdated: (updatedTask: Task) => void
    ) => {
      if (state.new) {
        (state.task as TaskNew).name = newName;
      } else {
        patchTask(
          {
            name: newName,
          },
          postUpdated
        );
      }
    };

    const updateSql = (
      newSql: string,
      postUpdated: (updatedTask: Task) => void
    ) => {
      if (state.new) {
        (state.task as TaskNew).sql = newSql;
      } else {
        patchTask(
          {
            sql: newSql,
          },
          postUpdated
        );
      }
    };

    const updateRollbackSql = (
      newSql: string,
      postUpdated: (updatedTask: Task) => void
    ) => {
      if (state.new) {
        (state.task as TaskNew).rollbackSql = newSql;
      } else {
        patchTask(
          {
            rollbackSql: newSql,
          },
          postUpdated
        );
      }
    };

    const updateDescription = (
      newDescription: string,
      postUpdated: (updatedTask: Task) => void
    ) => {
      if (state.new) {
        (state.task as TaskNew).description = newDescription;
      } else {
        patchTask(
          {
            description: newDescription,
          },
          postUpdated
        );
      }
    };

    const allowTransition = (transition: TaskStatusTransition): boolean => {
      const task: Task = state.task as Task;
      if (transition.type == "RESOLVE") {
        // if (pendingResolve(task)) {
        return allowResolve.value;
        // }
        // return false;
      }
      return true;
    };
    const tryStartTaskStatusTransition = (
      type: TaskStatusTransitionType,
      didTransit: () => {}
    ) => {
      const transition = TASK_STATUS_TRANSITION_LIST.get(type)!;
      updateStatusModalState.okText = transition.actionName;
      switch (transition.to) {
        case "OPEN":
          updateStatusModalState.style = "INFO";
          updateStatusModalState.title = "Reopen task?";
          break;
        case "DONE":
          updateStatusModalState.style = "SUCCESS";
          updateStatusModalState.title = "Resolve task?";
          break;
        case "CANCELED":
          updateStatusModalState.style = "INFO";
          updateStatusModalState.title = "Abort task?";
          break;
      }
      updateStatusModalState.payload = {
        transition,
        didTransit,
      };
      updateStatusModalState.show = true;
    };

    const doTaskStatusTransition = (
      payload: UpdateStatusModalStatePayload,
      outputValueList: string[],
      comment?: string
    ) => {
      let payloadChanged = false;
      for (let i = 0; i < outputValueList.length; i++) {
        const field = outputFieldList.value[i];
        if (!isEqual(state.task.payload[field.id], outputValueList[i])) {
          state.task.payload[field.id] = outputValueList[i];
          payloadChanged = true;
        }
      }

      const theComment = comment ? comment.trim() : undefined;
      patchTask(
        {
          status: payload.transition.to,
          comment: theComment ? theComment : undefined,
          payload: payloadChanged ? state.task.payload : undefined,
        },
        () => {
          payload.didTransit();
        }
      );
    };

    const updateAssigneeId = (newAssigneeId: PrincipalId) => {
      if (state.new) {
        (state.task as TaskNew).assigneeId = newAssigneeId;
      } else {
        patchTask({
          assigneeId: newAssigneeId,
        });
      }
    };

    const updateCustomField = (field: TaskField, value: any) => {
      console.log("updateCustomField", field.name, value);
      if (!isEqual(state.task.payload[field.id], value)) {
        state.task.payload[field.id] = value;
        if (!state.new) {
          patchTask({
            payload: state.task.payload,
          });
        }
      }
    };

    const doCreate = () => {
      store
        .dispatch("task/createTask", state.task)
        .then((createdTask) => {
          router.push(`/task/${taskSlug(createdTask.name, createdTask.id)}`);

          if (taskTemplate.value.type == "bytebase.database.schema.update") {
            store.dispatch("uistate/saveIntroStateByKey", {
              key: "table.create",
              newState: true,
            });
          }
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
            updaterId: currentUser.value.id,
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

    const changeStageStatus = (stage: StageProgressPatch, comment?: string) => {
      patchTask({
        stage,
        comment,
      });
    };

    const allowCreate = computed(() => {
      const newTask = state.task as TaskNew;
      if (isEmpty(newTask.name)) {
        return false;
      }

      if (!newTask.assigneeId) {
        return false;
      }

      if (newTaskTemplate.value.fieldList) {
        for (const field of newTaskTemplate.value.fieldList.filter(
          (item) => item.category == "INPUT"
        )) {
          if (
            field.type != "Switch" && // Switch is boolean value which always presents
            field.required &&
            field.isEmpty(state.task.payload[field.id])
          ) {
            return false;
          }
        }
      }
      return true;
    });

    const allowResolve = computed(() => {
      for (let i = 0; i < outputFieldList.value.length; i++) {
        const field = outputFieldList.value[i];
        if (field.required && field.isEmpty(state.task.payload[field.id])) {
          return false;
        }
      }
      return true;
    });

    // We may consider consolidating all the editing logic in one place. But for now
    // this controls editing "name, description and all custom fields".
    // On the other hand:
    // - Who can change task status is defined in a separate logic in this component
    // - Who can change stage status is defined in TaskStageFlow
    // - Who can reassign task is defined in TaskSidebar
    // - Who can change output field value is defined in TaskSidebar
    // - Anyone can comment
    // - Anyone can subscribe / unsubscribe
    const allowEditFields = computed(() => {
      // For now, we allow creator and assignee to update the field any time
      // when the task is OPEN. This may cause potential issue that the creator
      // might change some of the fields after the assignee follows the previous info
      // to deal the task. In the future, we could provide options to enforce more strict rules
      // e.g. disallow changing a particular field at a particular stage by a particular role.
      return (
        state.new ||
        ((state.task as Task).status == "OPEN" &&
          (currentUser.value.id == (state.task as Task).assignee?.id ||
            currentUser.value.id == (state.task as Task).creator.id))
      );
    });

    const showTaskStageFlowBar = computed(() => {
      return !state.new && state.task.stageList.length > 1;
    });

    const showTaskOutputPanel = computed(() => {
      return !state.new && outputFieldList.value.length > 0;
    });

    const showTaskSqlPanel = computed(() => {
      return (
        state.task.type == "bytebase.general" ||
        state.task.type == "bytebase.database.schema.update"
      );
    });

    const showTaskRollbackSqlPanel = computed(() => {
      return state.task.type == "bytebase.database.schema.update";
    });

    const applicableStatusTransitionList = computed(
      (): TaskStatusTransition[] => {
        const list: TaskStatusTransitionType[] = [];
        if (currentUser.value.id === (state.task as Task).assignee?.id) {
          list.push(
            ...ASSIGNEE_APPLICABLE_ACTION_LIST.get((state.task as Task).status)!
          );
        }
        if (currentUser.value.id === (state.task as Task).creator.id) {
          CREATOR_APPLICABLE_ACTION_LIST.get(
            (state.task as Task).status
          )!.forEach((item) => {
            if (list.indexOf(item) == -1) {
              list.push(item);
            }
          });
        }
        return list.map(
          (type: TaskStatusTransitionType) =>
            TASK_STATUS_TRANSITION_LIST.get(type)!
        );
      }
    );

    return {
      updateStatusModalState,
      state,
      updateName,
      updateDescription,
      updateSql,
      updateRollbackSql,
      allowTransition,
      tryStartTaskStatusTransition,
      doTaskStatusTransition,
      updateAssigneeId,
      updateCustomField,
      doCreate,
      changeStageStatus,
      allowCreate,
      currentUser,
      taskTemplate,
      outputFieldList,
      inputFieldList,
      allowEditFields,
      showTaskStageFlowBar,
      showTaskOutputPanel,
      showTaskSqlPanel,
      showTaskRollbackSqlPanel,
      applicableStatusTransitionList,
    };
  },
};
</script>
