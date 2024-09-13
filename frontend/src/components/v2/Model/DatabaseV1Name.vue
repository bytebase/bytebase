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
import { useRouter } from "vue-router";
import type { ComposedDatabase } from "@/types";
import { Engine, State } from "@/types/proto/v1/common";
import { autoDatabaseRoute } from "@/utils";

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

const router = useRouter();
const shouldShowLink = computed(() => {
  return props.link && props.database.syncState === State.ACTIVE;
});

const bindings = computed(() => {
  if (shouldShowLink.value) {
    return {
      to: autoDatabaseRoute(router, props.database),
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
  if (database.instanceResource.engine === Engine.REDIS) {
    return database.instanceResource.title;
  }
  return "";
});
</script>
