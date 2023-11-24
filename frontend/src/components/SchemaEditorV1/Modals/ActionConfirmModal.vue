<template>
  <BBModal
    :title="props.title"
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-80 mb-6">
      <p>{{ props.description }}</p>
    </div>
    <div class="w-full flex items-center justify-end mt-2 gap-x-3">
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
import { ButtonProps, NButton } from "naive-ui";
import { PropType } from "vue";

const props = defineProps({
  title: {
    type: String,
    default: "",
  },
  description: {
    type: String,
    default: "",
  },
  negativeButtonProps: {
    type: Object as PropType<ButtonProps>,
    default: () => ({ quaternary: true }),
  },
  positiveButtonProps: {
    type: Object as PropType<ButtonProps>,
    default: () => ({ type: "primary" }),
  },
});

const emit = defineEmits<{
  (event: "close"): void;
  (event: "confirm"): void;
}>();

const handleConfirmButtonClick = async () => {
  emit("confirm");
};

const dismissModal = () => {
  emit("close");
};
</script>
