<template>
  <!-- This div is used to insulate it from outside element to prevent resizing the BBAvatar -->
  <div>
    <div
      class="flex tracking-wide justify-center items-center select-none"
      :class="textClass"
      :style="[style]"
    >
      <slot>{{ initials }}</slot>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, PropType } from "vue";

import { hashCode } from "./BBUtil";

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

type SizeType = "small" | "normal" | "large" | "huge";

const sizeClassMap: Map<SizeType, string> = new Map([
  ["small", "w-6 h-6 text-xs font-normal"],
  ["normal", "w-8 h-8 text-sm font-normal"],
  ["large", "w-24 h-24 text-4xl font-medium"],
  ["huge", "w-36 h-36 text-5xl font-medium"],
]);

export default {
  name: "BBAvatar",
  props: {
    username: {
      type: String,
      default: "",
    },
    size: {
      type: String as PropType<SizeType>,
      default: "normal",
    },
    rounded: {
      type: Boolean,
      default: true,
    },
    backgroundColor: {
      type: String,
    },
  },
  setup(props, ctx) {
    const initials = computed(() => {
      let parts = props.username.split(/[ -]/);
      let initials = "";
      for (var i = 0; i < parts.length; i++) {
        for (var j = 0; j < parts[i].length; j++) {
          // Skip non-alphabet leading letters
          if (/[a-zA-Z]/.test(parts[i].charAt(j))) {
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
      };
    });

    const textClass = computed(() => {
      return sizeClassMap.get(props.size);
    });

    return {
      initials,
      textClass,
      style,
    };
  },
};
</script>
