<template>
  <NTooltip>
    <template #trigger>
      <template v-if="status === Advice_Level.SUCCESS">
        <CheckIcon :size="16" class="text-success" />
      </template>
      <template v-else-if="status === Advice_Level.WARNING">
        <TriangleAlertIcon :size="16" class="text-warning" />
      </template>
      <template v-else-if="status === Advice_Level.ERROR">
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
  CircleAlertIcon,
  PlayIcon,
  TriangleAlertIcon,
} from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

const props = defineProps<{
  status: Advice_Level;
}>();

const { t } = useI18n();

const statusMessage = computed(() => {
  switch (props.status) {
    case Advice_Level.SUCCESS:
      return "Success";
    case Advice_Level.WARNING:
      return "Warning";
    case Advice_Level.ERROR:
      return "Error";
    default:
      return t("issue.sql-check.not-executed-yet");
  }
});
</script>
