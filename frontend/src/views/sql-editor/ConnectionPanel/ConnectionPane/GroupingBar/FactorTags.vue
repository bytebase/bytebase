<template>
  <Draggable
    v-model="factorList"
    item-key="factor"
    animation="300"
    class="flex flex-col items-start gap-y-1.5 hide-scrollbar"
    ghost-class="opacity-50"
    draggable=".factor"
    handle=".handle"
    @change="handleDrag"
  >
    <template
      #item="{ element: sf, index }: { element: StatefulFactor; index: number }"
    >
      <FactorTag
        :key="index"
        :factor="sf"
        class="factor"
        style="--n-padding: 0 8px 0 2px; --n-icon-margin: 6px 2px 6px 0"
        @toggle-disabled="toggleDisabled(sf, index)"
        @remove="remove(sf, index)"
      >
        <template #icon>
          <GripVerticalIcon
            class="handle w-4 h-4 opacity-50 cursor-ns-resize"
          />
        </template>
      </FactorTag>
    </template>
    <template #footer>
      <AddFactorButton />
    </template>
  </Draggable>
</template>

<script lang="ts" setup>
import { GripVerticalIcon } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import Draggable from "vuedraggable";
import { useSQLEditorTreeStore } from "@/store";
import type { StatefulSQLEditorTreeFactor as StatefulFactor } from "@/types";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import AddFactorButton from "./AddFactorButton.vue";
import FactorTag from "./FactorTag.vue";

const treeStore = useSQLEditorTreeStore();
const { events } = useSQLEditorContext();
const { factorList } = storeToRefs(treeStore);

const toggleDisabled = (factor: StatefulFactor, _index: number) => {
  factor.disabled = !factor.disabled;
  treeStore.buildTree();
  events.emit("tree-ready");
};

const remove = (factor: StatefulFactor, index: number) => {
  factorList.value.splice(index, 1);
  treeStore.buildTree();
  events.emit("tree-ready");
};

const handleDrag = () => {
  treeStore.buildTree();
  events.emit("tree-ready");
};
</script>
