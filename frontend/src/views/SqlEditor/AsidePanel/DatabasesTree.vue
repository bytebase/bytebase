<template>
  <div
    class="databases-tree p-2 space-y-2 overflow-x-auto"
    v-if="!connectionContext.isLoadingTree"
  >
    <NTree block-line :data="connectionTree" />
  </div>
  <div v-else class="flex justify-center items-center h-full">
    <BBSpin title="Loading Databases..." />
  </div>
</template>

<script lang="ts" setup>
import { useNamespacedState } from "vuex-composition-helpers";

import type { SqlEditorState } from "../../../types";

const { connectionTree, connectionContext } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "connectionTree",
    "connectionContext",
  ]);
</script>
