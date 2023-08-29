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
import isChinese from "is-chinese";
import { computed, withDefaults } from "vue";
import { SYSTEM_BOT_EMAIL } from "@/types";
import { VueClass } from "@/utils";
import { hashCode } from "./BBUtil";
import { BBAvatarSizeType } from "./types";

const DEFAULT_BRANDING_COLOR = "#4f46e5";

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
  ["TINY", "0.5rem"], // customized font size
  ["SMALL", "0.675rem"], // customized font size
  ["NORMAL", "0.875rem"], // text-sm
  ["LARGE", "2.25rem"], // text-4xl
  ["HUGE", "3rem"], // text-5xl
]);

const props = withDefaults(
  defineProps<{
    username?: string;
    email?: string;
    size: BBAvatarSizeType;
    rounded?: boolean;
    backgroundColor?: string;
    overrideClass?: VueClass;
    overrideTextSize?: string;
  }>(),
  {
    username: "",
    email: "",
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

  if (props.email === SYSTEM_BOT_EMAIL) {
    return "BB";
  }

  // Priority
  // 1. First Chinese character
  // 2. At most the first 2 letters in email
  // Fallback if email is invalid
  // 1. The first and the last initial letter in title
  // 2. The first initial letter in title

  const chars = props.username.split("");
  for (let i = 0; i < chars.length; i++) {
    const ch = chars[i];
    if (isChinese(ch)) {
      return ch;
    }
  }

  if (props.email) {
    const nameInEmail = props.email.split("@")[0] ?? "";
    if (nameInEmail.length >= 2) {
      return nameInEmail.substring(0, 2).toUpperCase();
    }
    if (nameInEmail.length === 1) {
      return nameInEmail.charAt(0).toUpperCase();
    }
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
  if (props.email === SYSTEM_BOT_EMAIL) {
    return (
      getComputedStyle(document.documentElement)
        .getPropertyValue("--color-accent")
        .trim() || DEFAULT_BRANDING_COLOR
    );
  }
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
