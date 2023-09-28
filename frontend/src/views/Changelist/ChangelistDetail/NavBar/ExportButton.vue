<template>
  <ErrorTipsButton
    icon
    style="--n-padding: 0 10px"
    :errors="errors"
    :disabled="!allowEdit"
  >
    <template #icon>
      <heroicons:arrow-down-tray />
    </template>
  </ErrorTipsButton>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ErrorTipsButton } from "@/components/v2";
import { useChangelistDetailContext } from "../context";

const { t } = useI18n();
const { allowEdit, selectedChanges } = useChangelistDetailContext();

const errors = computed(() => {
  const errors: string[] = [];
  if (selectedChanges.value.length === 0) {
    errors.push(t("changelist.error.select-changes-to-export"));
  }
  return errors;
});
</script>
