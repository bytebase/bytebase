<template>
  <NButton
    v-if="advise"
    size="small"
    type="primary"
    @click="toggleGhost(advise.on)"
  >
    {{ advise.text() }}
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { PlanCheckDetailTableRow } from "@/components/PlanCheckRun/PlanCheckRunDetail.vue";

const props = defineProps<{
  row: PlanCheckDetailTableRow;
}>();

const emit = defineEmits<{
  (event: "toggle", on: boolean): void;
}>();

const { t } = useI18n();

const code = computed(() => {
  return props.row.checkResult.code;
});

const advise = computed(() => {
  if (!code.value) {
    return undefined;
  }
  if (code.value === 1801) {
    return {
      text: () => t("task.online-migration.enable"),
      on: true,
    };
  }
  if (code.value === 1803) {
    return {
      text: () => t("task.online-migration.disable"),
      on: false,
    };
  }
  return undefined;
});

const toggleGhost = (on: boolean) => {
  emit("toggle", on);
};
</script>
