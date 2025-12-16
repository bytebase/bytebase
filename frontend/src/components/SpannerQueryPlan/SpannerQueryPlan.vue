<template>
  <div class="spanner-query-plan">
    <div class="plan-header">
      <h3>Query Plan</h3>
      <div v-if="query" class="query-text">{{ query }}</div>
    </div>
    <div class="plan-tree">
      <SpannerPlanNode
        v-if="rootNode"
        :node="rootNode"
        :all-nodes="planNodes"
        :depth="0"
      />
      <div v-else class="no-plan">No query plan available</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import SpannerPlanNode from "./SpannerPlanNode.vue";
import type { SpannerPlanNodeData } from "./types";

const props = defineProps<{
  planSource: string;
  planQuery?: string;
}>();

const query = computed(() => props.planQuery);

const planNodes = computed((): SpannerPlanNodeData[] => {
  try {
    const parsed = JSON.parse(props.planSource);
    return parsed.planNodes || [];
  } catch {
    return [];
  }
});

const rootNode = computed((): SpannerPlanNodeData | undefined => {
  if (planNodes.value.length === 0) return undefined;
  // The first node (index 0) is the root node in Spanner query plans
  return planNodes.value.find((node) => node.index === 0);
});
</script>

<style scoped>
.spanner-query-plan {
  width: 100%;
  height: 100%;
  overflow: auto;
  padding: 16px;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen,
    Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif;
}

.plan-header {
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e0e0e0;
}

.plan-header h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.query-text {
  font-family: "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New",
    monospace;
  font-size: 13px;
  color: #666;
  background-color: #f5f5f5;
  padding: 8px 12px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-word;
}

.plan-tree {
  padding: 8px 0;
}

.no-plan {
  color: #999;
  font-style: italic;
  padding: 16px;
  text-align: center;
}
</style>
