<template>
  <BBModal
    :title="$t('schema-editor.default.expression')"
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <NInput
        ref="inputRef"
        v-model:value="state.expression"
        class="my-2"
        :autofocus="true"
      />
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3">
      <NButton @click="dismissModal">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton type="primary" @click="handleConfirmButtonClick">
        {{ $t("common.save") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { InputInst, NButton, NInput } from "naive-ui";
import { onMounted, reactive, ref } from "vue";
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

const inputRef = ref<InputInst>();
const notificationStore = useNotificationStore();
const state = reactive<LocalState>({
  expression: props.expression || "",
});

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

onMounted(() => {
  inputRef.value?.focus();
});
</script>
