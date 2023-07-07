<template>
  <NTooltip>
    <template #trigger>
      <NButton
        size="tiny"
        circle
        class="!ml-1"
        :loading="syncing"
        :tertiary="synced"
        :type="synced ? 'success' : 'default'"
        @click="syncNow"
      >
        <template #icon>
          <heroicons:arrow-path v-if="!synced" class="w-4 h-4" />
          <heroicons:check v-else class="w-4 h-4" />
        </template>
      </NButton>
    </template>

    <div class="whitespace-nowrap">
      {{ $t("common.sync-now") }}
    </div>
  </NTooltip>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { NButton, NTooltip } from "naive-ui";
import { useI18n } from "vue-i18n";

import { pushNotification } from "@/store";

const { t } = useI18n();
const syncing = ref(false);
const synced = ref(false);

const syncNow = async () => {
  if (syncing.value || synced.value) return;

  syncing.value = true;
  try {
    // await useIssueV1Store().fetchIssueByLegacyIssue(
    //   issue.value,
    //   true /* force */
    // );
    // TODO

    synced.value = true;
    // Show 'synced' status for several seconds to avoid user clicking sync
    // button too frequently.
    setTimeout(() => {
      synced.value = false;
    }, 5000);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.success"),
    });
  } finally {
    syncing.value = false;
  }
};
</script>
