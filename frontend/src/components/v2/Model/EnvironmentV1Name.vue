<template>
  <component
    :is="isLink ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="[isLink && !plain && 'normal-link', isLink && 'hover:underline']"
  >
    <NEllipsis :line-clamp="1">
      <span class="select-none" :class="textClass">
        {{ prefix }}
        <span v-if="isUnknown" class="text-gray-400 italic">
          NULL ENVIRONMENT
        </span>
        <HighlightLabelText
          v-else
          :text="environment.title"
          :keyword="keyword"
        />
        <slot name="suffix">
          {{ suffix }}
        </slot>
      </span>
    </NEllipsis>
    <ProductionEnvironmentV1Icon
      v-if="showIcon"
      :environment="environment"
      :class="iconClass ?? '!text-current'"
      :tooltip="tooltip"
    />
  </component>
</template>

<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { UNKNOWN_ENVIRONMENT_NAME } from "@/types";
import type { Environment } from "@/types/v1/environment";
import type { VueClass } from "@/utils";
import { autoEnvironmentRoute } from "@/utils";
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
    prefix?: string;
    showIcon?: boolean;
    textClass?: string;
    keyword?: string;
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
  }
);

const router = useRouter();

const isUnknown = computed(
  () => props.environment.name === UNKNOWN_ENVIRONMENT_NAME
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
</script>
