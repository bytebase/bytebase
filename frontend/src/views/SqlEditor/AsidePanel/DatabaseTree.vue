<template>
  <div
    v-if="!connectionContext.isLoadingTree"
    class="databases-tree p-2 space-y-2 h-full"
  >
    <div class="databases-tree--input">
      <NInput
        v-model:value="searchPattern"
        :placeholder="$t('sql-editor.search-databases')"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
    </div>
    <div class="databases-tree--tree overflow-y-auto">
      <NTree
        block-line
        leaf-only
        :data="connectionTree"
        :pattern="searchPattern"
        :default-expanded-keys="defaultExpanedKeys"
        :default-selected-keys="defaultSelectedKeys"
        :on-update:selected-keys="handleSelectedKeysChange"
      />
    </div>
  </div>
  <div v-else class="flex justify-center items-center h-full">
    <BBSpin :title="$t('sql-editor.loading-databases')" />
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
    return [`instance-${ctx.instanceId}`, `database-${ctx.databaseId}`];
  } else {
    return [];
  }
});

const defaultSelectedKeys = computed(() => {
  const ctx = connectionContext.value;
  if (ctx.hasSlug) {
    return [`database-${ctx.databaseId}`];
  } else {
    return [];
  }
});

const handleSelectedKeysChange = (
  keys: number[],
  options: Array<ConnectionAtom>
) => {
  const [selectedItem] = options;
  if (selectedItem.type === "table") {
    setConnectionContext({
      selectedDatabaseId: selectedItem.parentId,
      selectedTableName: selectedItem.label,
    });
  }
};
</script>

<style scoped>
.databases-tree--tree {
  height: calc(100% - 40px);
}
</style>
