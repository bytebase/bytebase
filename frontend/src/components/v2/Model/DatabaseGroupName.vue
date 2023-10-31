<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && !plain && 'normal-link'"
  >
    <span>{{ databaseGroup.databasePlaceholder }}</span>
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { ComposedDatabaseGroup } from "@/types";
import { projectV1Slug } from "@/utils";

const props = withDefaults(
  defineProps<{
    databaseGroup: ComposedDatabaseGroup;
    tag?: string;
    link?: boolean;
    plain?: boolean;
  }>(),
  {
    tag: "span",
    link: true,
    plain: false,
  }
);

const bindings = computed(() => {
  if (!props.databaseGroup) {
    return {};
  }

  const route = `/project/${projectV1Slug(
    props.databaseGroup.project
  )}/database-groups/${props.databaseGroup.databaseGroupName}`;
  if (props.link) {
    return {
      to: route,
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
