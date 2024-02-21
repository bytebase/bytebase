<template>
  <component
    :is="shouldShowLink ? 'router-link' : NEllipsis"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="[
      shouldShowLink && !plain && 'normal-link',
      shouldShowLink && 'hover:underline',
    ]"
  >
    <span v-if="prefix" class="mr-1 text-gray-400">{{ prefix }}</span>
    <span>{{ database.databaseName }}</span>
    <span
      v-if="showNotFound && database.syncState === State.DELETED"
      class="text-control-placeholder"
    >
      (NOT_FOUND)
    </span>
  </component>
</template>

<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { computed } from "vue";
import type { ComposedDatabase } from "@/types";
import { Engine, State } from "@/types/proto/v1/common";
import { databaseV1Url } from "@/utils";

const props = withDefaults(
  defineProps<{
    database: ComposedDatabase;
    link?: boolean;
    plain?: boolean;
    showNotFound?: boolean;
  }>(),
  {
    link: true,
    plain: false,
    showNotFound: false,
  }
);

const shouldShowLink = computed(() => {
  return props.link && props.database.syncState === State.ACTIVE;
});

const bindings = computed(() => {
  if (shouldShowLink.value) {
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
