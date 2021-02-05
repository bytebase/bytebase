<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
    <!-- Page header -->
    <div class="bg-white shadow">
      <PipelineHighlightPanel />
    </div>

    <!-- Flow -->
    <PipelineFlow />

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
              <PipelineContentBar />
              <PipelineSidebar class="mt-8 lg:hidden" />
              <div class="py-3 lg:pt-6 lg:pb-0">
                <PipelineContent />
              </div>
            </div>
            <section aria-labelledby="activity-title" class="mt-8 lg:mt-10">
              <PipelineActivityPanel />
            </section>
          </div>
          <PipelineSidebar class="hidden lg:block lg:pl-8" />
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import PipelineActivityPanel from "../views/PipelineActivityPanel.vue";
import PipelineFlow from "../views/PipelineFlow.vue";
import PipelineHighlightPanel from "../views/PipelineHighlightPanel.vue";
import PipelineContent from "../views/PipelineContent.vue";
import PipelineContentBar from "../views/PipelineContentBar.vue";
import PipelineSidebar from "../views/PipelineSidebar.vue";

export default {
  name: "PipelineDetail",
  components: {
    PipelineActivityPanel,
    PipelineContent,
    PipelineContentBar,
    PipelineFlow,
    PipelineHighlightPanel,
    PipelineSidebar,
  },
  setup(props, ctx) {},
};
</script>
