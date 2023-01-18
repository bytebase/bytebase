<template>
  <NTooltip trigger="hover" :delay="500">
    <template #trigger>
      <NButton quaternary size="tiny" v-bind="$attrs" @click="gotoAlterSchema">
        <heroicons-outline:pencil-alt class="w-4 h-4" />
      </NButton>
    </template>
    {{ $t("database.alter-schema") }}
  </NTooltip>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { stringify } from "qs";
import { NButton } from "naive-ui";

import { useRepositoryStore } from "@/store";
import type {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import type { Database, Repository } from "@/types";
import { baseDirectoryWebUrl } from "@/types";

const props = defineProps<{
  database: Database;
  schema?: SchemaMetadata;
  table?: TableMetadata;
}>();

const exampleSQL = computed(() => {
  const { schema, table } = props;
  if (schema && table) {
    if (schema.name) {
      return `ALTER TABLE ${schema.name}.${table.name}`;
    }
    return `ALTER TABLE ${table.name}`;
  }
  return `ALTER TABLE`;
});

const gotoAlterSchema = () => {
  const { database } = props;

  const { project } = database;
  if (project.workflowType === "VCS") {
    useRepositoryStore()
      .fetchRepositoryByProjectId(database.project.id)
      .then((repository: Repository) => {
        window.open(
          baseDirectoryWebUrl(repository, {
            DB_NAME: database.name,
            ENV_NAME: database.instance.environment.name,
            TYPE: "ddl",
          }),
          "_blank"
        );
      });
    return;
  }

  const query = {
    template: "bb.issue.database.schema.update",
    name: `[${database.name}] Alter schema`,
    project: database.project.id,
    databaseList: database.id,
    sql: exampleSQL.value,
  };
  const url = `/issue/new?${stringify(query)}`;
  window.open(url, "_blank");
};
</script>
