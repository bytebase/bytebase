<template>
  <NInput
    v-model:value="state.value"
    :disabled="disabled"
    :placeholder="$t('plugin.ai.text-to-sql-placeholder')"
    class="bb-ai-prompt-input"
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
import { onMounted, reactive } from "vue";
import { NInput } from "naive-ui";

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

onMounted(() => {
  requestAnimationFrame(() => {
    const input = document.querySelector(
      ".bb-ai-prompt-input input[type=text]"
    ) as HTMLInputElement;
    if (input) {
      input.focus();
    }
  });
});
</script>

<style lang="postcss">
.bb-ai-prompt-input .n-input__input-el {
  @apply !ring-0;
}
</style>
