<template>
  <NWatermark
    v-for="(line, i) in lines"
    :key="i"
    style="z-index: 10000000"
    :content="line"
    :cross="true"
    :fullscreen="true"
    :font-size="SIZE"
    :font-color="`rgba(128, 128, 128, .1)`"
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
import { NWatermark } from "naive-ui";
import { computed } from "vue";
import { featureToRef, useCurrentUserV1 } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { UNKNOWN_USER_NAME } from "@/types";
import { extractUserUID } from "@/utils";

const GAP = 320;
const SIZE = 16;
const PADDING = 6;

const currentUserV1 = useCurrentUserV1();
const setting = computed(() =>
  useSettingV1Store().getSettingByName("bb.workspace.watermark")
);
const hasWatermarkFeature = featureToRef("bb.feature.watermark");

const lines = computed(() => {
  const user = currentUserV1.value;
  const uid = extractUserUID(user.name);
  if (user.name === UNKNOWN_USER_NAME) return [];
  if (!hasWatermarkFeature.value) return [];
  if (setting.value?.value?.stringValue !== "1") return [];

  const lines: string[] = [];
  lines.push(`${user.title} (${uid})`);
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
