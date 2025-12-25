<template>
  <div>
    <div v-if="!disabled" class="textinfolabel mb-1">
      {{ $t('sql-review.input-then-press-enter') }}
    </div>
    <NDynamicTags
      :size="'large'"
      :disabled="disabled"
      :value="value"
      :input-props="{
        clearable: true,
      }"
      :input-style="'min-width: 20rem;'"
      @update:value="onUpdate"
    />
  </div>
</template>

<script lang="ts" setup>
import { NDynamicTags } from "naive-ui";
import type { RuleConfigComponent } from "@/types";

defineProps<{
  config: RuleConfigComponent;
  value: string[];
  disabled: boolean;
}>();

const emit = defineEmits<{
  (event: "update:value", value: string[]): void;
}>();

const onUpdate = (arr: string[]) => {
  emit(
    "update:value",
    arr.map((v) => v.trim()).filter((v) => v)
  );
};
</script>
