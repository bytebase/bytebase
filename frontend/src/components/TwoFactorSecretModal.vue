<template>
  <BBModal
    :title="$t('two-factor.your-two-factor-secret.self')"
    :subtitle="$t('two-factor.your-two-factor-secret.description')"
    class="outline-solid outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-auto mb-4 py-2">
      <code class="pr-4">{{ props.secret }}</code>
    </div>
    <div class="w-full flex items-center justify-end gap-x-3 pr-1 pb-1">
      <NButton @click="dismissModal">
        {{ $t("common.close") }}
      </NButton>
      <NButton v-if="isSupported" type="primary" @click="copySecret">
        {{ $t("common.copy") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
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

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});

const copySecret = () => {
  copyTextToClipboard(props.secret).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("two-factor.your-two-factor-secret.copy-succeed"),
    });
  });
  dismissModal();
};
</script>
