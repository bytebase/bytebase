<template>
  <!-- This div is used to insulate it from outside element to prevent resizing the BBAvatar -->
  <div>
    <div
      class="flex justify-center items-center select-none"
      :class="textClass"
      :style="[style]"
    >
      <slot>{{ initials }}</slot>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, withDefaults } from "vue";
import { VueClass } from "@/utils";
import { hashCode } from "./BBUtil";
import { BBAvatarSizeType } from "./types";

const BACKGROUND_COLOR_LIST: string[] = [
  "#64748B",
  "#EF4444",
  "#F97316",
  "#EAB308",
  "#84CC16",
  "#22C55E",
  "#10B981",
  "#06B6D4",
  "#0EA5E9",
  "#3B82F6",
  "#6366F1",
  "#8B5CF6",
  "#A855F7",
  "#D946EF",
  "#EC4899",
  "#F43F5E",
];

const sizeClassMap: Map<BBAvatarSizeType, string> = new Map([
  ["TINY", "w-5 h-5 font-medium"],
  ["SMALL", "w-6 h-6 font-medium"],
  ["NORMAL", "w-8 h-8 font-medium"],
  ["LARGE", "w-24 h-24 font-medium"],
  ["HUGE", "w-36 h-36 font-medium"],
]);

const fontStyleClassMap: Map<BBAvatarSizeType, string> = new Map([
  ["SMALL", "0.675rem"], // customized font size
  ["NORMAL", "0.875rem"], // text-sm
  ["LARGE", "2.25rem"], // text-4xl
  ["HUGE", "3rem"], // text-5xl
]);

const props = withDefaults(
  defineProps<{
    username?: string;
    size: BBAvatarSizeType;
    rounded?: boolean;
    backgroundColor?: string;
    overrideClass?: VueClass;
    overrideTextSize?: string;
  }>(),
  {
    username: "",
    size: "NORMAL",
    rounded: true,
    backgroundColor: "",
    overrideClass: undefined,
    overrideTextSize: undefined,
  }
);

const initials = computed(() => {
  if (props.username == "?") {
    return "?";
  }

  const parts = props.username.split(/[ -]/);
  let initials = "";
  for (let i = 0; i < parts.length; i++) {
    for (let j = 0; j < parts[i].length; j++) {
      // Skip non-alphabet leading letters
      if (/[a-zA-Z0-9]/.test(parts[i].charAt(j))) {
        initials += parts[i].charAt(j);
        break;
      }
    }
  }
  if (initials.length <= 2) {
    return initials.toUpperCase();
  }
  // If there are more than 2 initials, we will pick the first and last initial.
  // Displaying > 2 letters in a circle doesn't look good.
  return (initials[0] + initials[initials.length - 1]).toUpperCase();
});

const backgroundColor = computed(() => {
  return (
    props.backgroundColor ||
    BACKGROUND_COLOR_LIST[
      (hashCode(props.username) & 0xfffffff) % BACKGROUND_COLOR_LIST.length
    ]
  );
});

const style = computed(() => {
  return {
    borderRadius: props.rounded ? "50%" : 0,
    backgroundColor: backgroundColor.value,
    color: "white",
    "font-size": props.overrideTextSize ?? fontStyleClassMap.get(props.size),
  };
});

const textClass = computed(() => {
  return props.overrideClass ?? sizeClassMap.get(props.size);
});
</script>
