<template>
  <p class="text-sm text-gray-500 mb-4">
    {{ $t("two-factor.description") }}
    <LearnMoreLink url="doc-link" />
  </p>
  <BBStepTab
    :step-item-list="stepTabList"
    :allow-next="true"
    :finish-title="$t('common.confirm-and-add')"
  >
    <template #0>
      <!-- placeholder -->
    </template>
    <template #1>
      <!-- placeholder -->
    </template>
  </BBStepTab>
</template>

<script lang="ts" setup>
import { useI18n } from "vue-i18n";
import { BBStepTabItem } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";

const { t } = useI18n();

const stepTabList: BBStepTabItem[] = [
  { title: t("two-factor.setup-steps.setup-auth-app") },
  { title: t("two-factor.setup-steps.download-recovery-codes") },
];
</script>
