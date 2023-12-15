<template>
  <div
    v-if="showBanner"
    class="max-auto py-1 px-3 w-full flex flex-row justify-center flex-wrap text-center text-white font-medium"
    :class="[bgColor, bgColorHover]"
  >
    <a
      v-if="announcementLink !== ''"
      :href="announcementLink"
      target="_blank"
      class="hover:underline flex flex-row items-center"
    >
      <p class="px-1">{{ announcementText }}</p>
      <heroicons-solid:arrow-long-right class="mr-3 w-5 h-5" />
    </a>
    <p v-else>{{ announcementText }}</p>
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useSettingSWRStore } from "@/store";
import { Announcement_AlertLevel } from "@/types/proto/v1/setting_service";
import { urlfy } from "@/utils";

const { workspaceProfileSetting } = storeToRefs(useSettingSWRStore());

const showBanner = computed(() => {
  return announcementText.value !== "";
});

const bgColor = computed(() => {
  switch (workspaceProfileSetting.value?.announcement?.level) {
    case Announcement_AlertLevel.ALERT_LEVEL_INFO:
      return "bg-info";
    case Announcement_AlertLevel.ALERT_LEVEL_WARNING:
      return "bg-warning";
    case Announcement_AlertLevel.ALERT_LEVEL_CRITICAL:
      return "bg-error";
    default:
      return "bg-info";
  }
});

const bgColorHover = computed(() => {
  switch (workspaceProfileSetting.value?.announcement?.level) {
    case Announcement_AlertLevel.ALERT_LEVEL_INFO:
      return "hover:bg-info-hover";
    case Announcement_AlertLevel.ALERT_LEVEL_WARNING:
      return "hover:bg-warning-hover";
    case Announcement_AlertLevel.ALERT_LEVEL_CRITICAL:
      return "hover:bg-error-hover";
    default:
      return "hover:bg-info-hover";
  }
});

const announcementText = computed(() => {
  return workspaceProfileSetting.value?.announcement?.text ?? "";
});

const announcementLink = computed(() => {
  const link = workspaceProfileSetting.value?.announcement?.link ?? "";
  if (link.length === 0) {
    return link;
  }

  return urlfy(link);
});
</script>
