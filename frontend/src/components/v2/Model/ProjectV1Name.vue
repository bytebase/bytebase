<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && !plain && 'normal-link'"
  >
    <NPerformantEllipsis>
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span v-html="renderedProjectName" />
    </NPerformantEllipsis>
  </component>
</template>

<script lang="ts" setup>
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import type { Project } from "@/types/proto/v1/project_service";
import {
  autoProjectRoute,
  getHighlightHTMLByRegExp,
  projectV1Name,
} from "@/utils";

const props = withDefaults(
  defineProps<{
    project: Project;
    tag?: string;
    link?: boolean;
    plain?: boolean;
    hash?: string;
    keyword?: string;
  }>(),
  {
    tag: "span",
    link: true,
    plain: false,
    hash: "",
    keyword: "",
  }
);
const router = useRouter();

const bindings = computed(() => {
  if (props.link) {
    return {
      to: autoProjectRoute(router, props.project),
      activeClass: "",
      exactActiveClass: "",
      onClick: (e: MouseEvent) => {
        e.stopPropagation();
      },
    };
  }
  return {};
});

const renderedProjectName = computed(() => {
  return getHighlightHTMLByRegExp(projectV1Name(props.project), props.keyword);
});
</script>
