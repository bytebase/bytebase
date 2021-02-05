<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
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
      <div class="py-8 lg:py-10">
        <div
          class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 lg:max-w-full lg:grid lg:grid-cols-3"
        >
          <div class="lg:col-span-2 lg:pr-8 lg:border-r lg:border-gray-200">
            <div>
              <PipelineContentBar :pipeline="pipeline" />
              <PipelineSidebar class="mt-8 lg:hidden" :pipeline="pipeline" />
              <div class="py-3 lg:pt-6 lg:pb-0">
                <PipelineContent :pipeline="pipeline" />
              </div>
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
import { computed } from "vue";
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
    const pipeline = computed(() =>
      store.getters["pipeline/pipelineById"](props.pipelineId)
    );

    return { pipeline, humanize };
  },
};
</script>
