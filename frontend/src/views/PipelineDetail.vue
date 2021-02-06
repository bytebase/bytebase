<template>
  <div
    id="pipeline-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <!-- Page header -->
    <div class="bg-white">
      <PipelineHighlightPanel :pipeline="pipeline" />
    </div>

    <!-- Flow -->
    <PipelineFlow :pipeline="pipeline" />

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
              <PipelineContentBar v-if="false" :pipeline="pipeline" />
              <PipelineSidebar class="lg:hidden" :pipeline="pipeline" />
              <PipelineContent :pipeline="pipeline" />
            </div>
            <section aria-labelledby="activity-title" class="mt-8 lg:mt-10">
              <PipelineActivityPanel :pipeline="pipeline" />
            </section>
          </div>
          <PipelineSidebar
            class="hidden lg:block lg:pl-8"
            :pipeline="pipeline"
          />
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, onMounted } from "vue";
import { useStore } from "vuex";
import { humanize } from "../utils";
import PipelineActivityPanel from "../views/PipelineActivityPanel.vue";
import PipelineFlow from "../views/PipelineFlow.vue";
import PipelineHighlightPanel from "../views/PipelineHighlightPanel.vue";
import PipelineContent from "../views/PipelineContent.vue";
import PipelineContentBar from "../views/PipelineContentBar.vue";
import PipelineSidebar from "../views/PipelineSidebar.vue";

export default {
  name: "PipelineDetail",
  props: {
    pipelineId: {
      required: true,
      type: String,
    },
  },
  components: {
    PipelineActivityPanel,
    PipelineContent,
    PipelineContentBar,
    PipelineFlow,
    PipelineHighlightPanel,
    PipelineSidebar,
  },

  setup(props, ctx) {
    const store = useStore();

    onMounted(() => {
      // Always scroll to top, the scrollBehavior doesn't seem to work.
      // The hypothesis is that because the scroll bar is in the nested
      // route, thus setting the scrollBehavior in the global router
      // won't work.
      document.getElementById("pipeline-detail-top")!.scrollIntoView();
    });

    const pipeline = computed(() =>
      store.getters["pipeline/pipelineById"](props.pipelineId)
    );

    return { pipeline, humanize };
  },
};
</script>
