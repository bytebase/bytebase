<template>
  <NInput
    v-model:value="state.value"
    :disabled="disabled"
    :placeholder="$t('plugin.ai.text-to-sql-placeholder')"
    @keypress.enter="handlePressEnter"
  >
    <template #prefix>
      <heroicons-outline:sparkles class="w-4 h-4 text-accent" />
    </template>
    <template #suffix>
      <NButton
        :quaternary="true"
        :disabled="!state.value"
        type="primary"
        size="small"
        @click="handlePressEnter"
      >
        ⏎
      </NButton>
    </template>
  </NInput>
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import { reactive } from "vue";

type LocalState = {
  value: string;
};

withDefaults(
  defineProps<{
    disabled?: boolean;
  }>(),
  {
    disabled: false,
  }
);

const emit = defineEmits<{
  (event: "enter", value: string): void;
}>();

const state = reactive<LocalState>({
  value: "",
});

const applyValue = (value: string) => {
  state.value = "";
  emit("enter", value);
};

const handlePressEnter = () => {
  applyValue(state.value);
};
</script>

<style lang="postcss">
.bb-ai-prompt-input .n-input__input-el {
  @apply !ring-0;
}
</style>
