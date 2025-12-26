<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && !plain && 'normal-link'"
  >
    <NPerformantEllipsis :line-clamp="1" :tooltip="true">
      <HighlightLabelText :text="projectV1Name(project)" :keyword="keyword" />
    </NPerformantEllipsis>
  </component>
</template>

<script lang="ts" setup>
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { autoProjectRoute, projectV1Name } from "@/utils";
import HighlightLabelText from "./HighlightLabelText.vue";

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
</script>
