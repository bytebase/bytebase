<template>
  <div class="flex flex-row items-center gap-x-1 text-sm">
    <NodeCheckbox v-if="selectionEnabled" :node="node" />

    <template v-if="node.type === 'instance'">
      <InstanceV1EngineIcon :instance="node.instance" />
      <span class="text-gray-500">
        {{ environmentForInstanceNode(node).title }}
      </span>
    </template>
    <template v-if="node.type === 'database'">
      <DatabaseIcon class="w-4 h-auto text-gray-400" />
      <span class="text-gray-500">
        {{ node.db.effectiveEnvironmentEntity.title }}
      </span>
    </template>

    <template v-if="node.type === 'group'">
      <SchemaIcon
        v-if="node.group === 'table'"
        class="w-4 h-auto text-gray-400"
      />
      <ViewIcon v-if="node.group === 'view'" class="w-4 h-auto text-gray-400" />
      <ProcedureIcon
        v-if="node.group === 'procedure'"
        class="w-4 h-auto text-gray-400"
      />
      <FunctionIcon
        v-if="node.group === 'function'"
        class="w-4 h-auto text-gray-400"
      />
    </template>

    <div
      v-if="node.type === 'column'"
      class="w-4 h-4 inline-flex items-center justify-center"
    >
      <PrimaryKeyIcon v-if="isPrimaryKeyColumn" class="w-4 h-4" />
      <IndexIcon v-if="isIndexColumn" class="w-4! h-4! text-gray-500" />
    </div>

    <SchemaIcon
      v-if="node.type === 'schema'"
      class="w-4 h-auto text-gray-400"
    />
    <TableIcon v-if="node.type === 'table'" class="w-4 h-auto text-gray-400" />
    <ViewIcon v-if="node.type === 'view'" class="w-4 h-auto text-gray-400" />
    <ProcedureIcon
      v-if="node.type === 'procedure'"
      class="w-4 h-auto text-gray-400"
    />
    <FunctionIcon
      v-if="node.type === 'function'"
      class="w-4 h-auto text-gray-400"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  DatabaseIcon,
  FunctionIcon,
  IndexIcon,
  PrimaryKeyIcon,
  ProcedureIcon,
  SchemaIcon,
  TableIcon,
  ViewIcon,
} from "@/components/Icon";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import { useSchemaEditorContext } from "../context";
import type { TreeNode, TreeNodeForInstance } from "./common";
import NodeCheckbox from "./NodeCheckbox";

const props = defineProps<{
  node: TreeNode;
}>();

const { selectionEnabled } = useSchemaEditorContext();

const isPrimaryKeyColumn = computed(() => {
  const { node } = props;
  if (node.type !== "column") return false;

  const { table, column } = node.metadata;
  const pk = table.indexes.find((idx) => idx.primary);
  if (!pk) return false;
  return pk.expressions.includes(column.name);
});
const isIndexColumn = computed(() => {
  if (isPrimaryKeyColumn.value) return false;

  const { node } = props;
  if (node.type !== "column") return false;
  const { table, column } = node.metadata;
  return table.indexes.some((idx) => idx.expressions.includes(column.name));
});

const environmentForInstanceNode = (node: TreeNodeForInstance) => {
  return useEnvironmentV1Store().getEnvironmentByName(
    node.instance.environment ?? ""
  );
};
</script>
