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
    <BBTextField
      v-if="editable"
      v-model:value="inputValue"
      pattern="[a-z]+"
      :disabled="disabled"
      :placeholder="$t('sql-review.input-then-press-enter')"
      @keyup.enter="push($event)"
    />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
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

const inputValue = ref("");

const push = (_: Event) => {
  const array = [...props.value];
  const val = inputValue.value.trim();
  if (val) {
    if (!array.includes(val)) {
      array.push(val);
      inputValue.value = "";
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
