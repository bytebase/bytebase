<template>
  <span :class="['px-1 py-0.5 rounded text-xs whitespace-nowrap', colorClass]">
    {{ label }}
  </span>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useSettingV1Store } from "@/store";
import { operatorDisplayMap } from "./exemptionDataUtils";

const { t } = useI18n(); // NOSONAR: Vue composable, not React hook

const props = withDefaults(
  defineProps<{
    level?: number;
    operator?: string;
    noLimit?: boolean;
  }>(),
  {
    level: undefined,
    operator: "<=",
    noLimit: false,
  }
);

const settingStore = useSettingV1Store(); // NOSONAR

const levelTitle = computed(() => {
  if (props.level === undefined) return undefined;
  const config = settingStore.classification[0];
  return config?.levels?.find((l) => l.level === props.level)?.title;
});

// Color mapping follows ClassificationLevelBadge.vue
const bgColorList = [
  "bg-green-200", // level 1
  "bg-yellow-200", // level 2
  "bg-orange-300", // level 3
  "bg-amber-500", // level 4
  "bg-red-500", // level 5+
];

const label = computed(() => {
  if (props.noLimit) return t("project.masking-exemption.all-levels");
  if (props.level === undefined)
    return t("project.masking-exemption.all-levels");
  const op = operatorDisplayMap[props.operator] ?? props.operator;
  const title = levelTitle.value;
  return title ? `${op} ${props.level} (${title})` : `${op} ${props.level}`;
});

const colorClass = computed(() => {
  if (props.noLimit || props.level === undefined) {
    return "bg-gray-200 text-gray-600";
  }
  const idx = Math.min(props.level - 1, bgColorList.length - 1);
  const bg = bgColorList[Math.max(0, idx)] ?? "bg-gray-200";
  return props.level >= 4 ? `${bg} text-white` : bg;
});
</script>
