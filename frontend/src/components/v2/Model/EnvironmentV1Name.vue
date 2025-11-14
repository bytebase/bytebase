<template>
  <component
    :is="isLink ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="[isLink && !plain && 'normal-link', isLink && 'hover:underline']"
    :style="
      showColor && backgroundColorRgb
        ? {
            backgroundColor: `rgba(${backgroundColorRgb}, 0.1)`,
            borderTopColor: `rgb(${backgroundColorRgb})`,
            color: `rgb(${backgroundColorRgb})`,
            padding: '0 6px',
            borderRadius: '4px',
          }
        : {}
    "
  >
    <span class="select-none inline-block truncate" :class="textClass">
      <span v-if="isUnknown" class="text-gray-400 italic">
        {{ nullEnvironmentPlaceholder }}
      </span>
      <HighlightLabelText v-else :text="environment.title" :keyword="keyword" />
      <slot name="suffix">
        {{ suffix }}
      </slot>
    </span>
    <ProductionEnvironmentV1Icon
      v-if="showIcon"
      :environment="environment"
      :class="iconClass ?? 'text-current!'"
      :tooltip="tooltip"
    />
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { NULL_ENVIRONMENT_NAME, UNKNOWN_ENVIRONMENT_NAME } from "@/types";
import type { Environment } from "@/types/v1/environment";
import type { VueClass } from "@/utils";
import { autoEnvironmentRoute, hexToRgb } from "@/utils";
import HighlightLabelText from "./HighlightLabelText.vue";
import ProductionEnvironmentV1Icon from "./ProductionEnvironmentV1Icon.vue";

const props = withDefaults(
  defineProps<{
    environment: Environment;
    tag?: string;
    link?: boolean;
    plain?: boolean;
    iconClass?: VueClass;
    tooltip?: boolean;
    suffix?: string;
    showIcon?: boolean;
    textClass?: string;
    keyword?: string;
    showColor?: boolean;
    nullEnvironmentPlaceholder?: string; // Placeholder for null/unknown environment.
  }>(),
  {
    tag: "span",
    link: true,
    plain: false,
    iconClass: undefined,
    tooltip: false,
    suffix: "",
    prefix: "",
    showIcon: true,
    textClass: "",
    keyword: "",
    showColor: true,
    nullEnvironmentPlaceholder: "NULL ENVIRONMENT",
  }
);

const router = useRouter();

const isUnknown = computed(
  () =>
    props.environment.name === UNKNOWN_ENVIRONMENT_NAME ||
    props.environment.name === NULL_ENVIRONMENT_NAME
);

const isLink = computed(() => props.link && !isUnknown.value);

const bindings = computed(() => {
  if (isLink.value) {
    return {
      to: {
        ...autoEnvironmentRoute(router, props.environment),
      },
      activeClass: "",
      exactActiveClass: "",
      onClick: (e: MouseEvent) => {
        e.stopPropagation();
      },
    };
  }
  return {};
});

const backgroundColorRgb = computed(() => {
  if (!props.environment.color) {
    return "";
  }
  return hexToRgb(props.environment.color).join(", ");
});
</script>
