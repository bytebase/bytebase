<template>
  <NInput
    v-model:value="state.value"
    :disabled="disabled"
    :placeholder="$t('plugin.ai.text-to-sql-placeholder')"
    type="textarea"
    :autosize="{
      minRows: 1,
      maxRows: 10,
    }"
    :input-props="{
      style: 'box-shadow: none !important',
    }"
    @keypress.enter="handlePressEnter"
  >
    <template #prefix>
      <heroicons-outline:sparkles class="w-4 h-4 text-accent" />
    </template>
    <template #suffix>
      <NPopover placement="bottom" style="--n-padding: 4px 6px">
        <template #trigger>
          <NButton
            :quaternary="true"
            :disabled="!state.value"
            type="primary"
            size="small"
            @click="handlePressEnter()"
          >
            ⏎
          </NButton>
        </template>
        <template #default>
          <div class="text-sm">
            <p class="flex items-center gap-1">
              <span>{{ $t("plugin.ai.send") }}</span>
              <span>({{ keyboardShortcutStr("⏎") }})</span>
            </p>
            <p class="flex items-center gap-1">
              <span>{{ $t("plugin.ai.new-line") }}</span>
              <span>({{ keyboardShortcutStr("shift+⏎") }})</span>
            </p>
          </div>
        </template>
      </NPopover>
    </template>
  </NInput>
</template>

<script lang="ts" setup>
import { NButton, NInput, NPopover } from "naive-ui";
import { reactive } from "vue";
import { keyboardShortcutStr } from "@/utils";

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

const handlePressEnter = (e?: KeyboardEvent) => {
  if (e?.shiftKey) {
    return;
  }
  applyValue(state.value);
  e?.preventDefault();
};
</script>
