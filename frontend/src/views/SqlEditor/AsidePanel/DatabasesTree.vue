<template>
  <div
    class="databases-tree p-2 space-y-2 overflow-x-auto"
    v-if="!connectionContext.isLoadingTree"
  >
    <NInput v-model:value="searchPattern" placeholder="Search Databases" />
    <NTree
      block-line
      :data="connectionTree"
      :pattern="searchPattern"
      :default-expanded-keys="defaultExpanedKeys"
      :default-selected-keys="defaultSelectedKeys"
      :on-update:selected-keys="handleSelectedKeysChange"
    />
  </div>
  <div v-else class="flex justify-center items-center h-full">
    <BBSpin title="Loading Databases..." />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import {
  useNamespacedState,
  useNamespacedActions,
} from "vuex-composition-helpers";

import type {
  SqlEditorState,
  ConnectionAtom,
  SqlEditorActions,
} from "../../../types";

const searchPattern = ref();
const { connectionTree, connectionContext } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "connectionTree",
    "connectionContext",
  ]);
const { setConnectionContext } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setConnectionContext"]
);

const defaultExpanedKeys = computed(() => {
  const ctx = connectionContext.value;
  if (ctx.hasSlug) {
    return [ctx.instanceId, ctx.databaseId];
  } else {
    [];
  }
});

const defaultSelectedKeys = computed(() => {
  const ctx = connectionContext.value;
  if (ctx.hasSlug) {
    return [ctx.databaseId];
  } else {
    [];
  }
});

const handleSelectedKeysChange = (
  keys: number[],
  options: Array<ConnectionAtom>
) => {
  const [selectedItem] = options;
  if (selectedItem.type === "table") {
    setConnectionContext({
      selectedTableId: selectedItem.id,
    });
  }
};
</script>
