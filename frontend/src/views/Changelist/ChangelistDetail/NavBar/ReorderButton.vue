<template>
  <ErrorTipsButton
    icon
    style="--n-padding: 0 10px"
    :errors="errors"
    :disabled="!allowEdit"
  >
    <template #icon>
      <heroicons:arrows-up-down />
    </template>
  </ErrorTipsButton>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ErrorTipsButton } from "@/components/v2";
import { useChangelistDetailContext } from "../context";

const { t } = useI18n();
const { allowEdit, changelist } = useChangelistDetailContext();

const errors = computed(() => {
  const errors: string[] = [];
  if (changelist.value.changes.length === 0) {
    errors.push(t("changelist.error.add-at-least-one-change"));
  }
  return errors;
});
</script>
