<template>
  <div class="plan-node" :class="{ 'is-scalar': isScalar }">
    <div class="node-content" @click="toggleExpanded">
      <span class="expand-icon" v-if="hasChildren">
        <ChevronDownIcon v-if="expanded" class="w-4" />
        <ChevronUpIcon v-else class="w-4" />
      </span>
      <span v-else class="expand-icon-placeholder"></span>

      <span class="node-kind" :class="kindClass">{{ node.kind }}</span>
      <span class="node-name">{{ node.displayName }}</span>

      <span v-if="shortDescription" class="node-description">
        {{ shortDescription }}
      </span>
    </div>

    <div v-if="hasMetadata" class="node-metadata">
      <div
        v-for="(value, key) in displayMetadata"
        :key="key"
        class="metadata-item"
      >
        <span class="metadata-key">{{ key }}:</span>
        <span class="metadata-value">{{ formatValue(value) }}</span>
      </div>
    </div>

    <div v-if="expanded && hasChildren" class="node-children">
      <div
        v-for="childLink in relationalChildLinks"
        :key="childLink.childIndex"
        class="child-wrapper"
      >
        <div v-if="childLink.type" class="child-link-type">
          {{ childLink.type }}
        </div>
        <SpannerPlanNode
          :node="getChildNode(childLink.childIndex)"
          :all-nodes="allNodes"
          :depth="depth + 1"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChevronDownIcon, ChevronUpIcon } from "lucide-vue-next";
import { computed, ref } from "vue";
import type { SpannerChildLink, SpannerPlanNodeData } from "./types";

const props = defineProps<{
  node: SpannerPlanNodeData;
  allNodes: SpannerPlanNodeData[];
  depth: number;
}>();

const expanded = ref(true);

const isScalar = computed(() => props.node.kind === "SCALAR");

const kindClass = computed(() => {
  switch (props.node.kind) {
    case "RELATIONAL":
      return "kind-relational";
    case "SCALAR":
      return "kind-scalar";
    default:
      return "kind-unknown";
  }
});

const hasChildren = computed(() => {
  return relationalChildLinks.value.length > 0;
});

// Only show relational children in the tree
const relationalChildLinks = computed((): SpannerChildLink[] => {
  if (!props.node.childLinks) return [];
  return props.node.childLinks.filter((link) => {
    const childNode = props.allNodes.find((n) => n.index === link.childIndex);
    return childNode && childNode.kind === "RELATIONAL";
  });
});

const shortDescription = computed(() => {
  return props.node.shortRepresentation?.description;
});

const hasMetadata = computed(() => {
  return props.node.metadata && Object.keys(props.node.metadata).length > 0;
});

const displayMetadata = computed(() => {
  if (!props.node.metadata) return {};
  // Filter out some internal metadata keys
  const filtered: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(props.node.metadata)) {
    if (!key.startsWith("_")) {
      filtered[key] = value;
    }
  }
  return filtered;
});

const toggleExpanded = () => {
  if (hasChildren.value) {
    expanded.value = !expanded.value;
  }
};

const getChildNode = (index: number): SpannerPlanNodeData => {
  const child = props.allNodes.find((n) => n.index === index);
  if (!child) {
    return {
      index,
      kind: "UNKNOWN",
      displayName: `Unknown Node (${index})`,
    };
  }
  return child;
};

const formatValue = (value: unknown): string => {
  if (value === null || value === undefined) return "null";
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
};
</script>

<style scoped>
.plan-node {
  margin-left: 0;
  font-size: 14px;
}

.plan-node.is-scalar {
  display: inline;
}

.node-content {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.node-content:hover {
  background-color: #f0f4f8;
}

.expand-icon {
  width: 16px;
  font-size: 10px;
  color: #666;
  user-select: none;
}

.expand-icon-placeholder {
  width: 16px;
}

.node-kind {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 3px;
  text-transform: uppercase;
}

.kind-relational {
  background-color: #e3f2fd;
  color: #1565c0;
}

.kind-scalar {
  background-color: #f3e5f5;
  color: #7b1fa2;
}

.kind-unknown {
  background-color: #f5f5f5;
  color: #666;
}

.node-name {
  font-weight: 500;
  color: #333;
}

.node-description {
  font-family: "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New",
    monospace;
  font-size: 12px;
  color: #666;
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-metadata {
  margin-left: 40px;
  margin-top: 4px;
  padding: 8px 12px;
  background-color: #fafafa;
  border-radius: 4px;
  border-left: 3px solid #e0e0e0;
}

.metadata-item {
  display: flex;
  gap: 8px;
  font-size: 12px;
  line-height: 1.6;
}

.metadata-key {
  font-weight: 500;
  color: #666;
}

.metadata-value {
  font-family: "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New",
    monospace;
  color: #333;
  word-break: break-all;
}

.node-children {
  margin-left: 24px;
  padding-left: 16px;
  border-left: 2px solid #e0e0e0;
}

.child-wrapper {
  margin-top: 4px;
}

.child-link-type {
  font-size: 11px;
  color: #999;
  margin-left: 8px;
  margin-bottom: 2px;
  font-style: italic;
}
</style>
