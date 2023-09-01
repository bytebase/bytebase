<template>
  <NRadioGroup :value="level" @update:value="$emit('update:level', $event)">
    <NRadio
      v-for="item in [
        Announcement_AlertLevel.ALERTLEVEL_INFO,
        Announcement_AlertLevel.ALERTLEVEL_WARNING,
        Announcement_AlertLevel.ALERTLEVEL_CRITICAL,
      ]"
      :key="item"
      :value="item"
    >
      {{
        $t(
          `settings.general.workspace.announcement-alert-level.field.${AlertLevelToString(
            item
          ).toLowerCase()}`
        )
      }}
    </NRadio>
  </NRadioGroup>
</template>

<script setup lang="ts">
import { NRadio, NRadioGroup } from "naive-ui";
import { Announcement_AlertLevel } from "@/types/proto/v1/setting_service";

defineProps<{
  level?: Announcement_AlertLevel;
}>();

defineEmits<{
  (event: "update:level", level: Announcement_AlertLevel): void;
}>();

const AlertLevelToString = (level: Announcement_AlertLevel): string => {
  switch (level) {
    case Announcement_AlertLevel.ALERTLEVEL_INFO:
      return "INFO";
    case Announcement_AlertLevel.ALERTLEVEL_WARNING:
      return "WARNING";
    case Announcement_AlertLevel.ALERTLEVEL_CRITICAL:
      return "CRITICAL";
    default:
      return "INFO";
  }
};
</script>
