<template>
  <div>
    <div v-if="!disabled" class="textinfolabel mb-1">
      {{ $t('sql-review.input-then-press-enter') }}
    </div>
    <BBTextField
      v-model:value="inputValue"
      pattern="[a-z]+"
      :disabled="disabled"
      :placeholder="$t('sql-review.input-then-press-enter')"
      @keyup.enter="push($event)"
    />
    <div v-if="value.length > 0" class="flex flex-wrap gap-4 mt-4">
      <BBBadge
        v-for="(val, i) in value"
        :key="i"
        :text="`${val}`"
        :can-remove="!disabled"
        @remove="remove(val)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { BBBadge, BBTextField } from "@/bbkit";
import type { RuleConfigComponent } from "@/types";

const props = defineProps<{
  config: RuleConfigComponent;
  value: string[];
  disabled: boolean;
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
