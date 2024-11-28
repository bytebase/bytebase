<template>
  <BBModal
    :title="$t('settings.release.new-version-available')"
    @close="onClose"
  >
    <div class="min-w-0 md:min-w-400">
      <div>
        <p class="whitespace-pre-wrap">
          <i18n-t keypath="settings.release.new-version-content">
            <template #tag>
              <a
                class="font-bold underline"
                target="_blank"
                :href="actuatorStore.releaseInfo.latest?.html_url"
              >
                {{ actuatorStore.releaseInfo.latest?.tag_name }}
              </a>
            </template>
          </i18n-t>
        </p>
      </div>
      <div class="mt-7 flex justify-end space-x-2">
        <NButton @click="onClose">
          {{ $t("common.dismiss") }}
        </NButton>
        <NButton type="primary" @click="onClick">
          {{ $t("common.learn-more") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import { BBModal } from "@/bbkit";
import { useActuatorV1Store, useSubscriptionV1Store } from "@/store";

const emit = defineEmits(["cancel"]);

const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();

const link = computed(() => {
  if (subscriptionStore.isSelfHostLicense) {
    return "https://www.bytebase.com/docs/get-started/install/overview";
  }
  return subscriptionStore.purchaseLicenseUrl;
});

const onClick = () => {
  window.open(link.value, "_blank");
  onClose();
};

const onClose = () => {
  actuatorStore.releaseInfo.ignoreRemindModalTillNextRelease = true;
  emit("cancel");
};
</script>
