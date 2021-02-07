<template>
  <div
    id="task-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <!-- Page header -->
    <div class="bg-white">
      <TaskHighlightPanel :task="task" />
    </div>

    <!-- Flow -->
    <TaskFlow :task="task" />

    <!-- Main Content -->
    <main
      class="flex-1 relative overflow-y-auto focus:outline-none"
      tabindex="-1"
    >
      <div class="py-6">
        <div
          class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 lg:max-w-full lg:grid lg:grid-cols-3"
        >
          <div class="lg:col-span-2 lg:pr-8 lg:border-r lg:border-gray-200">
            <div>
              <TaskContentBar v-if="false" :task="task" />
              <TaskSidebar class="lg:hidden" :task="task" />
              <TaskContent :task="task" />
            </div>
            <section aria-labelledby="activity-title" class="mt-8 lg:mt-10">
              <TaskActivityPanel :task="task" />
            </section>
          </div>
          <TaskSidebar class="hidden lg:block lg:pl-8" :task="task" />
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, onMounted } from "vue";
import { useStore } from "vuex";
import { humanize } from "../utils";
import TaskActivityPanel from "../views/TaskActivityPanel.vue";
import TaskFlow from "../views/TaskFlow.vue";
import TaskHighlightPanel from "../views/TaskHighlightPanel.vue";
import TaskContent from "../views/TaskContent.vue";
import TaskContentBar from "../views/TaskContentBar.vue";
import TaskSidebar from "../views/TaskSidebar.vue";

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

    onMounted(() => {
      // Always scroll to top, the scrollBehavior doesn't seem to work.
      // The hypothesis is that because the scroll bar is in the nested
      // route, thus setting the scrollBehavior in the global router
      // won't work.
      document.getElementById("task-detail-top")!.scrollIntoView();
    });

    const task = computed(() => store.getters["task/taskById"](props.taskId));

    return { task, humanize };
  },
};
</script>
