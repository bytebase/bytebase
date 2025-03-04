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
    <!-- eslint-disable-next-line vue/no-v-html -->
    <span v-html="renderedDatabaseName" />
    <span
      v-if="showNotFound && database.state === State.DELETED"
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
import { autoDatabaseRoute, getHighlightHTMLByRegExp } from "@/utils";

const props = withDefaults(
  defineProps<{
    database: ComposedDatabase;
    link?: boolean;
    plain?: boolean;
    showNotFound?: boolean;
    keyword?: string;
  }>(),
  {
    link: true,
    plain: false,
    showNotFound: false,
    keyword: "",
  }
);

const router = useRouter();
const shouldShowLink = computed(() => {
  return props.link && props.database.state === State.ACTIVE;
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

const renderedDatabaseName = computed(() => {
  return getHighlightHTMLByRegExp(props.database.databaseName, props.keyword);
});
</script>
