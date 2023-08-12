<template>
  <div v-if="showBanner" :class="[`${bgColor}`]">
    <div
      class="mx-auto py-1 px-3 w-full flex flex-row items-center justify-center flex-wrap"
    >
      <div class="py-1 px-3 font-medium text-white flex flex-row items-center">
        <p>{{ announcementText }}</p>
      </div>
      <div
        v-if="announcementLink.length > 0"
        class="item-center py-1 px-3 font-medium text-white truncate"
      >
        <a
          :href="announcementLink"
          target="_blank"
          class="text-center underline"
        >
          <heroicons-solid:arrow-long-right class="mr-3 w-5 h-5" />
        </a>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useSettingV1Store } from "@/store";
import { Announcement_AlertLevel } from "@/types/proto/v1/setting_service";
import { urlfy } from "@/utils";

const settingV1Store = useSettingV1Store();

const showBanner = computed(() => {
  return announcementText.value.length > 0;
});

const bgColor = computed(() => {
  switch (settingV1Store.workspaceProfileSetting?.announcement?.level) {
    case Announcement_AlertLevel.ALERTLEVEL_INFO:
      return "bg-info";
    case Announcement_AlertLevel.ALERTLEVEL_WARNING:
      return "bg-warning";
    case Announcement_AlertLevel.ALERTLEVEL_ERROR:
      return "bg-error";
    default:
      return "bg-info";
  }
});

const announcementText = computed(() => {
  return settingV1Store.workspaceProfileSetting?.announcement?.text ?? "";
});

const announcementLink = computed(() => {
  const link = settingV1Store.workspaceProfileSetting?.announcement?.link ?? "";
  if (link.length === 0) {
    return link;
  }

  return urlfy(link);
});
</script>
