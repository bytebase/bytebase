<template>
  <div
    class="flex font-normal tracking-wide justify-center items-center select-none"
    :class="textClass"
    :style="[style]"
  >
    <slot>{{ initials }}</slot>
  </div>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import isEmpty from "lodash-es/isEmpty";

import { hashCode } from "./BBUtil";

const BACKGROUND_COLOR_LIST: string[] = [
  "#94A3B8",
  "#F87171",
  "#FB923C",
  "#FACC15",
  "#A3E635",
  "#4ADE80",
  "#34D399",
  "#22D3EE",
  "#60A5FA",
  "#818CF8",
  "#A78BFA",
  "#E879F9",
  "#F472B6",
  "#FB7185",
];

type SizeType = "normal" | "large";

const sizeClassMap: Map<SizeType, string> = new Map([
  ["normal", "w-8 h-8 text-sm"],
  ["large", "w-24 h-24 text-4xl"],
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
      initials = initials.substr(0, 3).toUpperCase();
      return initials;
    });

    const backgroundColor = computed(() => {
      return BACKGROUND_COLOR_LIST[
        (hashCode(props.username) & 0xfffffff) % BACKGROUND_COLOR_LIST.length
      ];
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
