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
      class="leading-6"
      :class="[
        factor.disabled && 'line-through',
        allowDisable && 'hover:line-through',
      ]"
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
  allowDisable: boolean;
}>();
const emit = defineEmits<{
  (event: "toggle-disabled"): void;
  (event: "remove"): void;
}>();
const treeStore = useSQLEditorTreeStore();
const { factorList, filteredFactorList } = storeToRefs(treeStore);

const allowRemove = computed(() => {
  const { factor } = props;
  // Always allow to remove the disabled factor
  if (factor.disabled) {
    return true;
  }

  // Otherwise, we only allow to remove the enabled factor if this is the only factor
  // or there exists other enabled factors
  return factorList.value.length === 1 || filteredFactorList.value.length >= 2;
});

const clickable = computed(() => {
  if (props.factor.disabled) return true;
  return props.allowDisable;
});

const handleClick = () => {
  if (!clickable.value) {
    return;
  }
  emit("toggle-disabled");
};
</script>
