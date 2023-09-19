<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && 'normal-link'"
  >
    <span>{{ projectName(project) }}</span>
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Project } from "@/types";
import { projectName, projectSlug } from "@/utils";

const props = withDefaults(
  defineProps<{
    project: Project;
    tag?: string;
    link?: boolean;
  }>(),
  {
    tag: "span",
    link: true,
  }
);

const bindings = computed(() => {
  if (props.link) {
    return {
      to: `/project/${projectSlug(props.project)}`,
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
