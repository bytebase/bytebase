<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && !plain && 'normal-link'"
  >
    <span v-if="prefix" class="ml-1 text-gray-400">{{ prefix }}</span>
    <span>{{ database.databaseName }}</span>
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { databaseV1Slug } from "@/utils";

const props = withDefaults(
  defineProps<{
    database: ComposedDatabase;
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
  if (props.link) {
    return {
      to: `/db/${databaseV1Slug(props.database)}`,
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
