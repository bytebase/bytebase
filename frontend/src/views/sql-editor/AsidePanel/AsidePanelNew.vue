<template>
  <div
    class="aside-panel-new w-[30vw] fixed z-[999] right-0 top-0 bottom-0 bg-white flex flex-col overflow-hidden"
  >
    <div class="h-[34px] bg-red-200 text-xs font-mono">
      {{ treeStore.factorList }}
    </div>
    <div class="primary-sidebar flex-1 overflow-auto">
      <DatabaseTreeNew />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useSQLEditorTreeStore } from "@/store/modules/sqlEditorTree";
import DatabaseTreeNew from "./DatabaseTreeNew.vue";

const treeStore = useSQLEditorTreeStore();
</script>
