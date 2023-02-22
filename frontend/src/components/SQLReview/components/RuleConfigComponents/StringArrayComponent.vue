<template>
  <div>
    <div class="flex flex-wrap gap-4 mb-4">
      <BBBadge
        v-for="(val, i) in value"
        :key="i"
        :text="`${val}`"
        :can-remove="!disabled && editable"
        @remove="remove(val)"
      />
    </div>
    <input
      v-if="editable"
      type="text"
      pattern="[a-z]+"
      :disabled="disabled"
      :class="[
        'shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md',
        disabled && 'cursor-not-allowed',
      ]"
      :placeholder="$t('sql-review.input-then-press-enter')"
      @keyup.enter="push($event)"
    />
  </div>
</template>

<script lang="ts" setup>
import type { RuleConfigComponent } from "@/types";

const props = defineProps<{
  config: RuleConfigComponent;
  value: string[];
  disabled: boolean;
  editable: boolean;
}>();

const emit = defineEmits<{
  (event: "update:value", value: string[]): void;
}>();

const push = (e: Event) => {
  const array = [...props.value];
  const input = e.target as HTMLInputElement;
  const val = input.value.trim();
  if (val) {
    if (!array.includes(val)) {
      array.push(val);
      input.value = "";
      emit("update:value", array);
    }
  }
};

const remove = (val: string) => {
  const array = [...props.value];
  const index = array.indexOf(val);
  if (index >= 0) {
    array.splice(index, 1);
    emit("update:value", array);
  }
};
</script>
