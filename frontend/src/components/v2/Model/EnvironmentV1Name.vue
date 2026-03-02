<template>
  <component
    :is="isLink ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="[isLink && !plain && 'normal-link', isLink && 'hover:underline']"
    :style="
      showColor && !isMissing && backgroundColorRgb
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
      <span v-if="isUnset" class="text-control-light italic">{{ t("common.unassigned") }}</span>
      <span v-else-if="isDeleted" class="text-control-light line-through">
        {{ environment.title }}
      </span>
      <HighlightLabelText v-else :text="environment.title" :keyword="keyword" />
      <slot name="suffix">
        {{ suffix }}
      </slot>
    </span>
    <ProductionEnvironmentV1Icon
      v-if="showIcon && !isMissing"
      :environment="environment"
      :class="iconClass ?? 'text-current!'"
      :tooltip="tooltip"
    />
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useEnvironmentV1Store } from "@/store";
import {
  formatEnvironmentName,
  isValidEnvironmentName,
  NULL_ENVIRONMENT_NAME,
  UNKNOWN_ENVIRONMENT_NAME,
} from "@/types";
import type { Environment } from "@/types/v1/environment";
import type { VueClass } from "@/utils";
import { hexToRgb } from "@/utils";
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
  }>(),
  {
    tag: "span",
    link: true,
    plain: false,
    iconClass: undefined,
    tooltip: false,
    suffix: "",
    showIcon: true,
    textClass: "",
    keyword: "",
    showColor: true,
  }
);

const { t } = useI18n();
const environmentStore = useEnvironmentV1Store();

const isUnset = computed(
  () =>
    props.environment.name === UNKNOWN_ENVIRONMENT_NAME ||
    props.environment.name === NULL_ENVIRONMENT_NAME
);

const isDeleted = computed(() => {
  if (isUnset.value) return false;
  const real = environmentStore.getEnvironmentByName(
    props.environment.name,
    false
  );
  return !isValidEnvironmentName(real.name);
});

const isMissing = computed(() => isUnset.value || isDeleted.value);

const isLink = computed(() => props.link && !isMissing.value);

const bindings = computed(() => {
  if (isLink.value) {
    return {
      to: {
        path: `/${formatEnvironmentName(props.environment.id)}`,
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
