<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && 'normal-link'"
  >
    <span v-if="prefix" class="ml-1 text-gray-400">{{ prefix }}</span>
    <span>{{ database.name }}</span>
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { databaseV1Url } from "@/utils";

const props = withDefaults(
  defineProps<{
    database: ComposedDatabase;
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
      to: databaseV1Url(props.database),
      activeClass: "",
      exactActiveClass: "",
      onClick: (e: MouseEvent) => {
        e.stopPropagation();
      },
    };
  }
  return {};
});

const prefix = computed(() => {
  const { database } = props;
  if (database.instanceEntity.engine === Engine.REDIS) {
    return database.instanceEntity.title;
  }
  return "";
});
</script>
