<template>
  <SlowQueryPanel
    v-if="database"
    :show-project-column="false"
    :show-environment-column="false"
    :show-instance-column="false"
    :show-database-column="false"
    :support-option-id-list="[]"
    :readonly-search-scopes="readonlyScopes"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SlowQueryPanel } from "@/components/SlowQuery";
import type { ComposedDatabase } from "@/types";
import {
  SearchScope,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
} from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const readonlyScopes = computed((): SearchScope[] => {
  return [
    {
      id: "environment",
      value: extractEnvironmentResourceName(
        props.database.effectiveEnvironment
      ),
    },
    {
      id: "instance",
      value: extractInstanceResourceName(props.database.instance),
    },
    {
      id: "database",
      value: `${props.database.databaseName}-${props.database.uid}`,
    },
  ];
});
</script>
