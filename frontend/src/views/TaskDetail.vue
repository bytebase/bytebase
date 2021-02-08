<template>
  <div
    id="task-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <!-- Page header -->
    <div class="bg-white">
      <TaskHighlightPanel
        :task="state.task"
        :enableButton="enableHighlightButton"
        @click-button="doClickHighlightPanelButton"
      />
    </div>

    <!-- Flow -->
    <TaskFlow v-if="!state.new" :task="state.task" />
    <div v-else class="border-t border-block-border" />

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
import { onMounted, reactive, inject } from "vue";
import { useStore } from "vuex";
import isEmpty from "lodash-es/isEmpty";
import { UserStateSymbol } from "../components/ProvideUser.vue";
import { humanize } from "../utils";
import TaskActivityPanel from "../views/TaskActivityPanel.vue";
import TaskFlow from "../views/TaskFlow.vue";
import TaskHighlightPanel from "../views/TaskHighlightPanel.vue";
import TaskContent from "../views/TaskContent.vue";
import TaskContentBar from "../views/TaskContentBar.vue";
import TaskSidebar from "../views/TaskSidebar.vue";
import { User, Task } from "../types";
import { taskTemplateList, TaskField } from "../plugins";

interface LocalState {
  new: boolean;
  task: Task;
}

export default {
  name: "TaskDetail",
  props: {
    taskId: {
      required: true,
      type: String,
    },
  },
  components: {
    TaskActivityPanel,
    TaskContent,
    TaskContentBar,
    TaskFlow,
    TaskHighlightPanel,
    TaskSidebar,
  },

  setup(props, ctx) {
    const store = useStore();

    const currentUser = inject<User>(UserStateSymbol);

    const state = reactive<LocalState>({
      new: props.taskId.toLowerCase() == "new",
      task:
        props.taskId.toLowerCase() == "new"
          ? {
              type: "task",
              attributes: {
                name: "New Task",
                status: "PENDING",
                type: "bytebase.datasource.create",
                content: "Need a new database",
                currentStageId: "1",
                stageProgressList: [
                  {
                    stageId: "1",
                    stageName: "Request Database",
                    status: "CREATED",
                  },
                ],
                creator: {
                  id: currentUser!.id,
                  name: currentUser!.attributes.name,
                },
                assignee: {
                  id: currentUser!.id,
                  name: currentUser!.attributes.name,
                },
                payload: {
                  1: "mydb",
                },
              },
            }
          : JSON.parse(
              JSON.stringify(store.getters["task/taskById"](props.taskId))
            ),
    });

    const template = taskTemplateList.find(
      (template) => template.type == state.task.attributes.type
    );

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
      console.log("Create", state.task);
    };

    const doClickHighlightPanelButton = (buttonIndex: number) => {
      if (state.new) {
        // Create
        if (buttonIndex == 0) {
          doCreate();
        }
      }
    };

    const enableHighlightButton = (buttonIndex: number): boolean => {
      if (state.new) {
        // Create
        if (buttonIndex == 0) {
          if (template) {
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

    return {
      state,
      humanize,
      updateField,
      doClickHighlightPanelButton,
      enableHighlightButton,
    };
  },
};
</script>
