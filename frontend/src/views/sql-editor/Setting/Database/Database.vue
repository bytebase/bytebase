<template>
  <div class="w-full h-full flex flex-col gap-4 py-4 px-2 overflow-y-auto">
    <DatabaseDashboard
      :open-in-new-window="true"
      :on-click-database="handleClickDatabase"
    />
  </div>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import { SQL_EDITOR_DATABASE_MODULE } from "@/router/sqlEditor";
import {
  DEFAULT_PROJECT_NAME,
  defaultProject,
  type ComposedDatabase,
} from "@/types";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
  hasProjectPermissionV2,
  isDatabaseV1Queryable,
} from "@/utils";
import DatabaseDashboard from "@/views/DatabaseDashboard.vue";

const router = useRouter();
const allowQueryDatabase = (db: ComposedDatabase) => {
  if (db.project === DEFAULT_PROJECT_NAME) {
    return hasProjectPermissionV2(defaultProject(), "bb.databases.query");
  }
  return isDatabaseV1Queryable(db);
};
const handleClickDatabase = (db: ComposedDatabase) => {
  if (!allowQueryDatabase(db)) return;
  const projectName = extractProjectResourceName(db.project);
  const { instanceName, databaseName } = extractDatabaseResourceName(db.name);
  router.push({
    name: SQL_EDITOR_DATABASE_MODULE,
    params: {
      project: projectName,
      instance: instanceName,
      database: databaseName,
    },
  });
};
</script>
