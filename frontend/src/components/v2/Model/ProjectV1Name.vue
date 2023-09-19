<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && !plain && 'normal-link'"
  >
    <span>{{ projectV1Name(project) }}</span>
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Project } from "@/types/proto/v1/project_service";
import { projectV1Name, projectV1Slug } from "@/utils";

const props = withDefaults(
  defineProps<{
    project: Project;
    tag?: string;
    link?: boolean;
    plain?: boolean;
    hash?: string;
  }>(),
  {
    tag: "span",
    link: true,
    plain: false,
    hash: "",
  }
);

const bindings = computed(() => {
  if (props.link) {
    return {
      to: `/project/${projectV1Slug(props.project)}${props.hash}`,
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
