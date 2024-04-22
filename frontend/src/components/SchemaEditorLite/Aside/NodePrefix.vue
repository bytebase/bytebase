<template>
  <div class="flex flex-row items-center gap-x-1 text-sm">
    <NodeCheckbox v-if="selectionEnabled" :node="node" />

    <template v-if="node.type === 'instance'">
      <InstanceV1EngineIcon :instance="node.instance" />
      <span class="text-gray-500">
        {{ node.instance.environmentEntity.title }}
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
      <ProcedureIcon
        v-if="node.group === 'procedure'"
        class="w-4 h-auto text-gray-400"
      />
      <FunctionIcon
        v-if="node.group === 'function'"
        class="w-4 h-auto text-gray-400"
      />
    </template>

    <SchemaIcon
      v-if="node.type === 'schema'"
      class="w-4 h-auto text-gray-400"
    />
    <TableIcon v-if="node.type === 'table'" class="w-4 h-auto text-gray-400" />
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
import {
  DatabaseIcon,
  FunctionIcon,
  ProcedureIcon,
  SchemaIcon,
  TableIcon,
} from "@/components/Icon";
import type { InstanceV1EngineIcon } from "@/components/v2";
import { useSchemaEditorContext } from "../context";
import NodeCheckbox from "./NodeCheckbox";
import type { TreeNode } from "./common";

defineProps<{
  node: TreeNode;
}>();

const { selectionEnabled } = useSchemaEditorContext();
</script>
