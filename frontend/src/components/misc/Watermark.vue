<template>
  <NWatermark
    v-for="(line, i) in lines"
    :key="i"
    style="z-index: 10000000"
    :content="line"
    :cross="true"
    :fullscreen="true"
    :font-size="SIZE"
    :line-height="SIZE"
    :width="GAP"
    :height="GAP"
    :x-offset="0"
    :y-offset="calcYOffset(i)"
    :rotate="-15"
    :debug="false"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NWatermark } from "naive-ui";

import { featureToRef, useCurrentUser, useSettingByName } from "@/store";
import { UNKNOWN_ID } from "@/types";

const GAP = 320;
const SIZE = 16;
const PADDING = 6;

const currentUser = useCurrentUser();
const setting = useSettingByName("bb.workspace.watermark");
const hasWatermarkFeature = featureToRef("bb.feature.watermark");

const lines = computed(() => {
  const user = currentUser.value;
  if (user.id === UNKNOWN_ID) return [];
  if (!hasWatermarkFeature.value) return [];
  if (setting.value?.value !== "1") return [];

  const lines: string[] = [];
  lines.push(`${user.name} (${user.id})`);
  if (user.email) {
    lines.push(user.email);
  }

  return lines;
});

const calcYOffset = (i: number) => {
  const total = lines.value.length;
  const base = GAP - PADDING;
  const offset = (total - i) * (SIZE * 1.25);
  return base - offset;
};
</script>
