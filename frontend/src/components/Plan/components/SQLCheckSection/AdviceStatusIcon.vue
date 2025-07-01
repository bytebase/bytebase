<template>
  <NTooltip>
    <template #trigger>
      <template v-if="status === Advice_Status.SUCCESS">
        <CheckIcon :size="16" class="text-success" />
      </template>
      <template v-else-if="status === Advice_Status.WARNING">
        <TriangleAlertIcon :size="16" class="text-warning" />
      </template>
      <template v-else-if="status === Advice_Status.ERROR">
        <CircleAlertIcon :size="16" class="text-error" />
      </template>
      <template v-else>
        <PlayIcon :size="16" />
      </template>
    </template>
    {{ statusMessage }}
  </NTooltip>
</template>

<script setup lang="ts">
import {
  CheckIcon,
  PlayIcon,
  TriangleAlertIcon,
  CircleAlertIcon,
} from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Advice_Status } from "@/types/proto-es/v1/sql_service_pb";

const props = defineProps<{
  status: Advice_Status;
}>();

const { t } = useI18n();

const statusMessage = computed(() => {
  switch (props.status) {
    case Advice_Status.SUCCESS:
      return "Success";
    case Advice_Status.WARNING:
      return "Warning";
    case Advice_Status.ERROR:
      return "Error";
    default:
      return t("issue.sql-check.not-executed-yet");
  }
});
</script>
