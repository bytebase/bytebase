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
    <HighlightLabelText :text="extractDatabaseResourceName(database.name).databaseName" :keyword="keyword" />
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
import { State } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { autoDatabaseRoute, extractDatabaseResourceName } from "@/utils";
import HighlightLabelText from "./HighlightLabelText.vue";

const props = withDefaults(
  defineProps<{
    database: Database;
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

const shouldShowLink = computed(() => {
  return props.link && props.database.state === State.ACTIVE;
});

const bindings = computed(() => {
  if (shouldShowLink.value) {
    return {
      to: autoDatabaseRoute(props.database),
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
