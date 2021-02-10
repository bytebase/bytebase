<template>
  <div
    id="task-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <!-- Highlight Panel -->
    <div class="bg-white">
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
          v-else-if="currentUser.id === state.task.attributes.creator.id"
        >
          <template v-if="pendingState() !== 'CLOSED'">
            <button
              type="button"
              class="btn-normal px-4 py-2"
              @click.prevent="doAbort"
              :disabled="pendingState() === 'RUNNING'"
            >
              Abort
            </button>
          </template>

          <template v-if="pendingState() === 'RESOLVE'">
            <button
              type="button"
              class="btn-success px-4 py-2"
              @click.prevent="doResolve"
            >
              Resolve
            </button>
          </template>
          <template v-else-if="pendingState() === 'RUNNING'">
            <button
              type="button"
              class="btn-primary px-4 py-2"
              @click.prevent="doAbort"
            >
              Cancel
            </button>
          </template>
          <template v-else-if="pendingState() === 'CLOSED'">
            <button
              type="button"
              class="btn-normal px-4 py-2"
              @click.prevent="doReopen"
            >
              Reopen
            </button>
          </template>
        </template>
        <!-- Action Buttons only applicable to assignee -->
        <template
          v-else-if="currentUser.id === state.task.attributes.assignee?.id"
        >
          <template v-if="pendingState() === 'APPROVAL'">
            <button
              type="button"
              class="btn-primary px-4 py-2"
              @click.prevent="doApprove"
            >
              Approve
            </button>
          </template>
          <template v-else-if="pendingState() === 'RESOLVE'">
            <button
              v-if="
                currentUser.id === state.task.attributes.assignee?.id ||
                currentUser.id === state.task.attributes.creator.id
              "
              type="button"
              class="btn-success px-4 py-2"
              @click.prevent="doResolve"
            >
              Resolve
            </button>
          </template>
          <template v-else-if="pendingState() === 'RUNNING'">
            <button
              type="button"
              class="btn-primary px-4 py-2"
              @click.prevent="doAbort"
            >
              Cancel
            </button>
          </template>
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
        <div class="flex max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 lg:max-w-full">
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
import { humanize, idFromSlug, taskSlug } from "../utils";
import TaskHighlightPanel from "../views/TaskHighlightPanel.vue";
import TaskFlow from "../views/TaskFlow.vue";
import TaskOutputPanel from "../views/TaskOutputPanel.vue";
import TaskContentBar from "../views/TaskContentBar.vue";
import TaskContent from "../views/TaskContent.vue";
import TaskActivityPanel from "../views/TaskActivityPanel.vue";
import TaskSidebar from "../views/TaskSidebar.vue";
import { User, Task, TaskNew, TaskPatch } from "../types";
import { taskTemplateList, TaskField } from "../plugins";

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

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]();
    });

    const currentUser = inject<User>(UserStateSymbol);

    const refreshState = () => {
      const generalTaskTemplate = taskTemplateList.find(
        (template) => template.type == "bytebase.general"
      )!;
      return {
        new: props.taskSlug.toLowerCase() == "new",
        task:
          props.taskSlug.toLowerCase() == "new"
            ? generalTaskTemplate.buildTask({
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

    const doAbort = () => {
      patchTask({
        status: "CANCELED",
      });
    };

    const doApprove = () => {};

    const doReopen = () => {
      patchTask({
        status: "OPEN",
      });
    };

    const doResolve = () => {
      patchTask({
        status: "DONE",
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

    const pendingState = (): "APPROVAL" | "RESOLVE" | "RUNNING" | "CLOSED" => {
      if (
        state.task.attributes.status === "DONE" ||
        state.task.attributes.status === "CANCELED"
      ) {
        return "CLOSED";
      } else {
        const currentStage = (state.task as Task).attributes.stageProgressList.find(
          (stage) => stage.id == state.task.attributes.currentStageId
        );
        if (currentStage) {
          if (currentStage.type === "SIMPLE") {
            return "RESOLVE";
          } else if (currentStage.type === "ENVIRONMENT") {
            if (state.task.attributes.status === "OPEN") {
              return "RUNNING";
            }
            return "APPROVAL";
          }
        }
        return "CLOSED";
      }
    };

    return {
      state,
      template,
      humanize,
      updateField,
      doCreate,
      doAbort,
      doApprove,
      doReopen,
      doResolve,
      enableHighlightButton,
      pendingState,
      currentUser,
    };
  },
};
</script>
