<template>
  <div
    class="border px-2 relative rounded-sm group bg-white text-control"
    :class="[
      clickable ? 'cursor-pointer hover:bg-gray-100' : 'cursor-not-allowed',
      factor.disabled ? 'opacity-50' : '',
    ]"
    @click.stop="handleClick"
  >
    <span
      class="leading-6 hover:line-through"
      :class="factor.disabled && 'line-through'"
    >
      {{ readableSQLEditorTreeFactor(factor.factor) }}
    </span>
    <button
      v-if="allowRemove"
      class="hidden group-hover:flex bg-gray-100 absolute rounded-full top-0 right-0 hover:bg-gray-300 z-10 translate-x-[50%] translate-y-[-40%] w-4 h-4 items-center justify-center"
      @click.stop="$emit('remove')"
    >
      <heroicons:x-mark class="w-3.5 h-3.5" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useSQLEditorTreeStore } from "@/store/modules/sqlEditorTree";
import {
  StatefulSQLEditorTreeFactor as StatefulFactor,
  readableSQLEditorTreeFactor,
} from "@/types";

const props = defineProps<{
  factor: StatefulFactor;
}>();
const emit = defineEmits<{
  (event: "toggle-disabled"): void;
  (event: "remove"): void;
}>();
const treeStore = useSQLEditorTreeStore();
const { factorList, filteredFactorList } = storeToRefs(treeStore);

const allowDisable = computed(() => {
  // Disallow to disable the only one enabled factor
  return filteredFactorList.value.length > 1;
});

const allowRemove = computed(() => {
  if (factorList.value.length <= 1) {
    // Disallow to remove the only one factor
    return false;
  }
  const { factor } = props;
  if (!factor.disabled) {
    if (filteredFactorList.value.length <= 1) {
      // Disallow to remove the only one enabled factor
      return false;
    }
  }
  return true;
});

const clickable = computed(() => {
  if (props.factor.disabled) return true;
  return allowDisable.value;
});

const handleClick = () => {
  if (!clickable.value) {
    return;
  }
  emit("toggle-disabled");
};
</script>
