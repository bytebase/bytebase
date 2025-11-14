<template>
  <NInput
    ref="inputRef"
    v-model:value="state.value"
    :disabled="disabled"
    :placeholder="$t('plugin.ai.text-to-sql-placeholder')"
    :autosize="{
      minRows: 1,
      maxRows: 10,
    }"
    :input-props="{
      style: 'box-shadow: none !important',
    }"
    size="small"
    type="textarea"
    style="--n-padding-left: 6px; --n-padding-right: 0px"
    @keypress.enter="handlePressEnter"
  >
    <template #suffix>
      <NPopover placement="bottom" style="--n-padding: 6px 8px">
        <template #trigger>
          <NButton
            :quaternary="true"
            :disabled="!state.value"
            type="primary"
            size="small"
            style="--n-padding: 0 6px"
            @click="handlePressEnter()"
          >
            ⏎
          </NButton>
        </template>
        <template #default>
          <div class="text-xs flex flex-col gap-1">
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
import { type InputInst, NButton, NInput, NPopover } from "naive-ui";
import { onMounted, reactive, ref, watch } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { keyboardShortcutStr, nextAnimationFrame } from "@/utils";
import { useAIContext } from "../logic";

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

const { pendingPreInput, events } = useAIContext();
const inputRef = ref<InputInst>();

const applyValue = (value: string) => {
  state.value = "";
  emit("enter", value);
};

const handlePressEnter = (e?: KeyboardEvent) => {
  if (e?.shiftKey) {
    return;
  }
  e?.preventDefault();
  if (!state.value.trim()) return;
  applyValue(state.value);
};

onMounted(() => {
  inputRef.value?.focus();
});

watch(
  pendingPreInput,
  async () => {
    await nextAnimationFrame();
    if (!pendingPreInput.value) return;
    state.value = pendingPreInput.value;
    pendingPreInput.value = undefined;
  },
  {
    immediate: true,
    flush: "post",
  }
);

useEmitteryEventListener(events, "new-conversation", () => {
  inputRef.value?.focus();
});
</script>
