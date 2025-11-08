<template>
  <BBModal
    v-model:show="modalVisible"
    :title="$t('common.warning')"
    :show-close="true"
    :close-on-esc="true"
    @close="handleClose"
  >
    <div
      class="w-120 max-w-full pt-2 pb-4 text-control wrap-break-word text-sm"
    >
      <div v-if="resources.length === 0">
        {{ $t("resource.delete-warning", { name: target }) }}
      </div>
      <div v-else class="flex flex-col gap-y-2">
        <p>
          {{
            description ||
            $t("resource.delete-warning-with-resources", {
              name: target,
            })
          }}
        </p>
        <ul class="list-disc">
          <Resource
            v-for="(resource, i) in resources"
            :key="i"
            :show-prefix="true"
            :link="true"
            :resource="resource"
          />
        </ul>
        <p v-if="!description">{{ $t("resource.delete-warning-retry") }}</p>
      </div>
    </div>

    <div class="pt-4 border-t border-block-border flex justify-end gap-x-3">
      <NButton size="small" @click.prevent="handleClose">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton
        v-if="showPositiveButton"
        type="warning"
        size="small"
        data-label="bb-modal-confirm-button"
        @click.prevent="handleSubmit"
      >
        {{ $t("common.continue-anyway") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="tsx" setup>
import { NButton } from "naive-ui";
import { ref } from "vue";
import { BBModal } from "@/bbkit";
import Resource from "./Resource.vue";

defineProps<{
  target: string;
  description?: string;
  resources: string[];
  showPositiveButton: boolean;
}>();

const emit = defineEmits<{
  (event: "on-submit"): void;
  (event: "on-close"): void;
}>();

const modalVisible = ref(false);

const open = () => {
  modalVisible.value = true;
};

const handleClose = () => {
  modalVisible.value = false;
  emit("on-close");
};

const handleSubmit = () => {
  modalVisible.value = false;
  emit("on-submit");
};

defineExpose({ open });
</script>
