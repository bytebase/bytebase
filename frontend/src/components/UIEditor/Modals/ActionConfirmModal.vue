<template>
  <BBModal
    :title="props.title"
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-80 mb-2">
      <p>{{ props.description }}</p>
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3 pr-1 pb-1">
      <button type="button" class="btn-normal" @click="dismissModal">
        {{ $t("common.cancel") }}
      </button>
      <button class="btn-primary" @click="handleConfirmButtonClick">
        {{ $t("common.confirm") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { PropType } from "vue";

const props = defineProps({
  title: {
    type: String as PropType<string>,
    default: "",
  },
  description: {
    type: String as PropType<string>,
    default: "",
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
