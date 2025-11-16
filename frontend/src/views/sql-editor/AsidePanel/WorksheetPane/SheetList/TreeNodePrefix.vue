<template>
  <FileCodeIcon
    v-if="node.worksheet"
    class="w-4 h-auto text-gray-600"
  />
  <FolderOpenIcon
    v-else-if="expandedKeys.has(node.key)"
    class="w-4 h-auto text-gray-600"
  />
  <template v-else-if="node.key === rootPath">
    <FolderPenIcon
      v-if="view === 'draft'"
      class="w-4 h-auto text-gray-600"
    />
    <FolderSyncIcon
      v-else-if="view === 'shared'"
      class="w-4 h-auto text-gray-600"
    />
    <FolderCodeIcon
      v-else
      class="w-4 h-auto text-gray-600"
    />
  </template>
  <FolderMinusIcon
    v-else-if="node.empty"
    class="w-4 h-auto text-gray-600"
  />
  <FolderCodeIcon
    v-else
    class="w-4 h-auto text-gray-600"
  />
</template>

<script setup lang="ts">
import {
  FileCodeIcon,
  FolderCodeIcon,
  FolderMinusIcon,
  FolderOpenIcon,
  FolderPenIcon,
  FolderSyncIcon,
} from "lucide-vue-next";
import type {
  SheetViewMode,
  WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";

defineProps<{
  node: WorksheetFolderNode;
  expandedKeys: Set<string>;
  rootPath: string;
  view: SheetViewMode;
}>();
</script>
