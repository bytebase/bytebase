<template>
  <NWatermark
    style="z-index: 10000000"
    :content="version"
    cross
    fullscreen
    :font-size="16"
    :font-color="`rgba(256, 128, 128, .003)`"
    :line-height="16"
    :width="128"
    :height="128"
    :x-offset="24"
    :y-offset="80"
    :rotate="15"
  />
  <NWatermark
    v-for="(line, i) in lines"
    :key="i"
    style="z-index: 10000001"
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
import {
  extractUserId,
  featureToRef,
  useActuatorV1Store,
  useCurrentUserV1,
} from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { UNKNOWN_USER_NAME } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

const GAP = 320;
const SIZE = 16;
const PADDING = 6;

const currentUserV1 = useCurrentUserV1();
const version = computed(
  () =>
    useActuatorV1Store().version +
    "-" +
    useActuatorV1Store().gitCommitBE.substring(0, 7)
);
const settingStore = useSettingV1Store();
const hasWatermarkFeature = featureToRef(PlanFeature.FEATURE_WATERMARK);

const lines = computed(() => {
  const user = currentUserV1.value;
  const uid = extractUserId(user.name);
  if (user.name === UNKNOWN_USER_NAME) return [];
  if (!hasWatermarkFeature.value) return [];
  if (!settingStore.workspaceProfileSetting?.watermark) return [];

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
