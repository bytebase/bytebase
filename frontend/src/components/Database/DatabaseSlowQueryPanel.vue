<template>
  <SlowQueryPanel
    v-if="database"
    :key="`slow-query.${database.name}`"
    :show-project-column="false"
    :show-environment-column="false"
    :show-instance-column="false"
    :show-database-column="false"
    :support-option-id-list="[]"
    :readonly-search-scopes="readonlyScopes"
    v-bind="$attrs"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SlowQueryPanel } from "@/components/SlowQuery";
import type { ComposedDatabase } from "@/types";
import type { SearchScope } from "@/utils";
import {
  extractEnvironmentResourceName,
  extractProjectResourceName,
} from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const readonlyScopes = computed((): SearchScope[] => {
  return [
    {
      id: "project",
      value: extractProjectResourceName(props.database.project),
    },
    {
      id: "environment",
      value: extractEnvironmentResourceName(
        props.database.effectiveEnvironment
      ),
    },
    {
      id: "database",
      value: props.database.name,
    },
  ];
});
</script>
