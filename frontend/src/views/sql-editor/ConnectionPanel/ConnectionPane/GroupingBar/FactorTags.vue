<template>
  <div class="flex flex-row items-start flex-wrap gap-x-1.5 gap-y-1.5">
    <FactorTag
      v-for="(factor, i) in factorList"
      :key="i"
      :factor="factor"
      @toggle-disabled="toggleDisabled(factor, i)"
      @remove="remove(factor, i)"
    />
    <AddFactorButton />
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { useSQLEditorTreeStore } from "@/store";
import type { StatefulSQLEditorTreeFactor as StatefulFactor } from "@/types";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import AddFactorButton from "./AddFactorButton.vue";
import FactorTag from "./FactorTag.vue";

const treeStore = useSQLEditorTreeStore();
const { events } = useSQLEditorContext();
const { factorList } = storeToRefs(treeStore);

const toggleDisabled = (factor: StatefulFactor, index: number) => {
  factor.disabled = !factor.disabled;
  treeStore.buildTree();
  events.emit("tree-ready");
};

const remove = (factor: StatefulFactor, index: number) => {
  factorList.value.splice(index, 1);
  treeStore.buildTree();
  events.emit("tree-ready");
};
</script>
