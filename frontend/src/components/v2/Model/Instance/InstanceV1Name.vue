<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && !plain && 'normal-link'"
  >
    <InstanceV1EngineIcon
      v-if="icon && iconPosition === 'prefix'"
      :instance="instance"
    />

    <slot name="prefix" />

    <span class="line-clamp-1" :class="textClass">{{
      instanceV1Name(instance)
    }}</span>

    <InstanceV1EngineIcon
      v-if="icon && iconPosition === 'suffix'"
      :instance="instance"
    />
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { Instance } from "@/types/proto/v1/instance_service";
import { instanceV1Name, instanceV1Slug } from "@/utils";
import InstanceV1EngineIcon from "./InstanceV1EngineIcon.vue";

const props = withDefaults(
  defineProps<{
    instance: Instance;
    tag?: string;
    link?: boolean;
    icon?: boolean;
    plain?: boolean;
    iconPosition?: "prefix" | "suffix";
    textClass?: string;
  }>(),
  {
    tag: "span",
    link: true,
    icon: true,
    plain: false,
    iconPosition: "prefix",
    textClass: "",
  }
);

const bindings = computed(() => {
  if (props.link) {
    return {
      to: `/instance/${instanceV1Slug(props.instance)}`,
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
