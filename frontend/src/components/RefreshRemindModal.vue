<template>
  <BBModal :title="$t('refresh-remind.title')" @close="onClose">
    <div class="min-w-0 md:min-w-400">
      <div>
        <p class="whitespace-pre-wrap">
          {{ $t("refresh-remind.description") }}
        </p>
      </div>
      <div class="mt-7 flex justify-end space-x-2">
        <NButton @click="onClose">
          {{ $t("common.dismiss") }}
        </NButton>
        <NButton type="primary" @click="onRefresh">
          {{ $t("common.refresh") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { BBModal } from "@/bbkit";

const emit = defineEmits(["cancel"]);

const onRefresh = () => {
  window.location.reload();
};

const onClose = () => {
  emit("cancel");
};
</script>
