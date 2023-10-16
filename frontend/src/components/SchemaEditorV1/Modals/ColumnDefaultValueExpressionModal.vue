<template>
  <BBModal
    :title="$t('schema-editor.default.expression')"
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <BBTextField
        class="my-2 w-full"
        :value="state.expression"
        @input="handleExpressionChange"
      />
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3 pr-1 pb-1">
      <button type="button" class="btn-normal" @click="dismissModal">
        {{ $t("common.cancel") }}
      </button>
      <button class="btn-primary" @click="handleConfirmButtonClick">
        {{ $t("common.save") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { useNotificationStore } from "@/store";

interface LocalState {
  expression: string;
}

const props = defineProps<{
  expression?: string;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "update:expression", value: string): void;
}>();

const notificationStore = useNotificationStore();
const state = reactive<LocalState>({
  expression: props.expression || "",
});

const handleExpressionChange = (event: Event) => {
  state.expression = (event.target as HTMLInputElement).value;
};

const handleConfirmButtonClick = async () => {
  if (!state.expression) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Expression cannot be empty",
    });
    return;
  }

  emit("update:expression", state.expression);
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
