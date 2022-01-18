<template>
  <div class="w-80 flex flex-col justify-start items-start">
    <div class="w-full leading-8">
      <p>{{ props.content }}</p>
    </div>
    <div class="w-full mt-4 flex justify-end space-x-2">
      <NButton @click="handleColse">{{ $t("common.close") }}</NButton>
      <NButton type="primary" @click="handleConfirm">{{
        $t("common.confirm")
      }}</NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { defineEmits, defineProps } from "vue";

const props = defineProps({
  content: {
    type: String,
    default: "",
  },
});

const emit = defineEmits<{
  (e: "confirm"): void;
  (e: "close"): void;
}>();

const handleConfirm = () => {
  emit("confirm");
};

const handleColse = () => {
  emit("close");
};
</script>

<style scoped></style>
