<template>
  <NRadioGroup
    :value="level"
    :disabled="!allowEdit"
    @update:value="$emit('update:level', $event)"
  >
    <NRadio
      v-for="item in [
        Announcement_AlertLevel.INFO,
        Announcement_AlertLevel.WARNING,
        Announcement_AlertLevel.CRITICAL,
      ]"
      :key="item"
      :value="item"
    >
      {{
        $t(
          `dynamic.settings.general.workspace.announcement-alert-level.field.${AlertLevelToString(
            item
          ).toLowerCase()}`
        )
      }}
    </NRadio>
  </NRadioGroup>
</template>

<script setup lang="ts">
import { NRadio, NRadioGroup } from "naive-ui";
import { Announcement_AlertLevel } from "@/types/proto-es/v1/setting_service_pb";

defineProps<{
  level?: Announcement_AlertLevel;
  allowEdit: boolean;
}>();

defineEmits<{
  (event: "update:level", level: Announcement_AlertLevel): void;
}>();

const AlertLevelToString = (level: Announcement_AlertLevel): string => {
  switch (level) {
    case Announcement_AlertLevel.INFO:
      return "INFO";
    case Announcement_AlertLevel.WARNING:
      return "WARNING";
    case Announcement_AlertLevel.CRITICAL:
      return "CRITICAL";
    default:
      return "INFO";
  }
};
</script>
