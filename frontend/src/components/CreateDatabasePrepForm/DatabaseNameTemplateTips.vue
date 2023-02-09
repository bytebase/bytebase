<template>
  <div v-if="mismatch" class="text-warning space-y-1 mt-2 text-xs">
    <p>{{ $t("database.doesnt-match-database-name-template") }}</p>
    <p>
      <code>{{ project.dbNameTemplate }}</code>
    </p>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import type { DatabaseLabel, Project } from "@/types";
import { buildDatabaseNameRegExpByTemplate } from "@/utils";

const props = defineProps<{
  name: string;
  project: Project;
  labelList: DatabaseLabel[];
}>();

const mismatch = computed(() => {
  const { project, name } = props;
  if (!name) {
    // Don't be too noisy
    return false;
  }

  const { dbNameTemplate } = project;
  if (!dbNameTemplate) {
    return false;
  }

  const regex = buildDatabaseNameRegExpByTemplate(dbNameTemplate);
  return !regex.test(name);
});
</script>
