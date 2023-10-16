<template>
  <div class="px-0.5">
    <NCheckbox
      :checked="checked"
      :disabled="disabled"
      @update:checked="handleToggleChecked"
    >
      {{ readableSQLEditorTreeFactor(factor) }}
    </NCheckbox>
  </div>
</template>
<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useSQLEditorTreeStore } from "@/store/modules/sqlEditorTree";
import {
  SQLEditorTreeFactor as Factor,
  readableSQLEditorTreeFactor,
} from "@/types";

const props = defineProps<{
  factor: Factor;
}>();
const emit = defineEmits<{
  (event: "toggle", on: boolean): void;
}>();

const treeStore = useSQLEditorTreeStore();
const { factorList } = storeToRefs(treeStore);

const allowRemove = computed(() => {
  // Not allowed to remove the only one factor
  return factorList.value.length > 1;
});
const checked = computed(() => {
  return factorList.value.findIndex((sf) => sf.factor === props.factor) >= 0;
});
const disabled = computed(() => {
  if (checked.value) {
    if (!allowRemove.value) return true;
  }
  return false;
});

const handleToggleChecked = (on: boolean) => {
  if (disabled.value) return;
  if (on === checked.value) return;
  emit("toggle", on);
};
</script>
