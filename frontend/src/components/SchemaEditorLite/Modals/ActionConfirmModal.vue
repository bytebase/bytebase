<template>
  <BBModal
    :title="props.title"
    :show="show"
    class="shadow-inner outline-solid outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-80 mb-6">
      <p>{{ props.description }}</p>
    </div>
    <div class="w-full flex items-center justify-end mt-2 gap-x-2">
      <NButton v-bind="negativeButtonProps" @click="dismissModal">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton v-bind="positiveButtonProps" @click="handleConfirmButtonClick">
        {{ $t("common.confirm") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import type { ButtonProps } from "naive-ui";
import { NButton } from "naive-ui";
import { BBModal } from "@/bbkit";

const props = withDefaults(
  defineProps<{
    show: boolean;
    title: string;
    description: string;
    negativeButtonProps?: ButtonProps;
    positiveButtonProps?: ButtonProps;
  }>(),
  {
    show: true,
    title: "",
    description: "",
    negativeButtonProps: () => ({ quaternary: true }),
    positiveButtonProps: () => ({ type: "primary" }),
  }
);

const emit = defineEmits<{
  (event: "close"): void;
  (event: "confirm"): void;
  (event: "update:show", show: boolean): void;
}>();

const handleConfirmButtonClick = async () => {
  emit("confirm");
  emit("update:show", false);
};

const dismissModal = () => {
  emit("close");
  emit("update:show", false);
};
</script>
