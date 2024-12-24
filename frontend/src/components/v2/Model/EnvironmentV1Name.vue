<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="[link && !plain && 'normal-link', link && 'hover:underline']"
  >
    <span class="line-clamp-1 select-none" :class="textClass">
      {{ prefix }}
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span v-html="renderedEnvironmentName" />
      <slot name="suffix">
        {{ suffix }}
      </slot>
    </span>
    <ProductionEnvironmentV1Icon
      v-if="showIcon"
      :environment="environment"
      :class="iconClass ?? '!text-current'"
      :tooltip="tooltip"
    />
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import type { Environment } from "@/types/proto/v1/environment_service";
import type { VueClass } from "@/utils";
import {
  autoEnvironmentRoute,
  environmentV1Name,
  getHighlightHTMLByRegExp,
} from "@/utils";
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

const bindings = computed(() => {
  if (props.link) {
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

const renderedEnvironmentName = computed(() => {
  return getHighlightHTMLByRegExp(
    environmentV1Name(props.environment),
    props.keyword
  );
});
</script>
