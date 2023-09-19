<template>
  <BBModal
    :title="$t('two-factor.your-two-factor-secret.self')"
    :subtitle="$t('two-factor.your-two-factor-secret.description')"
    class="outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-auto mb-4 py-2">
      <code class="pr-4">{{ props.secret }}</code>
    </div>
    <div class="w-full flex items-center justify-end space-x-3 pr-1 pb-1">
      <button type="button" class="btn-cancel" @click="dismissModal">
        {{ $t("common.close") }}
      </button>
      <button class="btn-primary" @click="copySecret">
        {{ $t("common.copy") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

const props = defineProps({
  secret: {
    type: String,
    default: "",
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();

const dismissModal = () => {
  emit("close");
};

const copySecret = () => {
  toClipboard(props.secret).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("two-factor.your-two-factor-secret.copy-succeed"),
    });
  });
  dismissModal();
};
</script>
