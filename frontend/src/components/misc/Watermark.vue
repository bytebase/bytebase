<template>
  <NWatermark
    v-if="watermark"
    style="z-index: 10000000"
    :content="watermark"
    :cross="true"
    :fullscreen="true"
    :font-size="16"
    :line-height="16"
    :width="384"
    :height="384"
    :x-offset="12"
    :y-offset="60"
    :rotate="-15"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NWatermark } from "naive-ui";

import { useCurrentUser, useSettingByName } from "@/store";
import { UNKNOWN_ID } from "@/types";

const currentUser = useCurrentUser();
const setting = useSettingByName("bb.workspace.watermark");

const watermark = computed(() => {
  if (currentUser.value.id === UNKNOWN_ID) return "";
  if (setting.value?.value !== "1") return "";
  return `${currentUser.value.name} (${currentUser.value.id})`;
});
</script>
