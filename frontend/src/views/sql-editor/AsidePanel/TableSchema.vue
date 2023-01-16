<template>
  <div v-if="schema && table" class="table-schema">
    <div class="table-schema--header">
      <div class="table-schema--header-title mr-1 flex items-center">
        <heroicons-outline:table class="h-4 w-4 mr-1" />
        <span v-if="schema.name" class="font-semibold">{{ schema.name }}.</span>
        <span class="font-semibold">{{ table.name }}</span>
      </div>
      <div
        class="table-schema--header-actions flex-1 flex justify-end space-x-2"
      >
        <div class="action-edit flex items-center">
          <NTooltip trigger="hover">
            <template #trigger>
              <NButton text @click="gotoAlterSchema">
                <heroicons-outline:pencil-alt class="w-4 h-4" />
              </NButton>
            </template>
            {{ $t("database.alter-schema") }}
          </NTooltip>
        </div>
        <div class="action-close flex items-center">
          <NTooltip trigger="hover">
            <template #trigger>
              <NButton text @click="handleClosePane">
                <heroicons-outline:x class="w-4 h-4" />
              </NButton>
            </template>
            {{ $t("sql-editor.close-pane") }}
          </NTooltip>
        </div>
      </div>
    </div>
    <div class="table-schema--meta text-gray-500 text-sm">
      <div class="pb-1">
        <span class="mr-1">{{ $t("database.row-count-est") }}</span>
        <span>{{ table.rowCount }}</span>
      </div>
      <div class="flex justify-between items-center w-full text-xs py-2">
        <div class="table-schema--content-column">
          <span>Columns</span>
        </div>
        <div class="table-schema--content-column">
          <span>Data Type</span>
        </div>
      </div>
    </div>
    <div class="table-schema--content text-sm text-gray-400 overflow-y-auto">
      <div
        v-for="(column, index) in table.columns"
        :key="index"
        class="flex justify-between items-center w-full p-1 hover:bg-link-hover"
      >
        <div class="table-schema--content-column text-gray-600">
          <span>{{ column.name }}</span>
        </div>
        <div class="table-schema--content-column">
          <span>{{ column.type }}</span>
        </div>
      </div>
    </div>
  </div>
  <div v-else class="h-full flex justify-center items-center">
    {{ $t("sql-editor.table-schema-placeholder") }}
  </div>
</template>

<script lang="ts" setup>
import { isUndefined } from "lodash-es";
import { computed } from "vue";
import { stringify } from "qs";
import type { Repository } from "@/types";
import { baseDirectoryWebUrl } from "@/types";
import {
  useConnectionTreeStore,
  useDatabaseStore,
  useDBSchemaStore,
  useRepositoryStore,
} from "@/store";

const emit = defineEmits<{
  (e: "close-pane"): void;
}>();

const connectionTreeStore = useConnectionTreeStore();
const databaseStore = useDatabaseStore();
const dbSchemaStore = useDBSchemaStore();
const tableAtom = computed(() => connectionTreeStore.selectedTableAtom);
const schema = computed(() => {
  const atom = tableAtom.value;
  if (isUndefined(atom)) {
    return undefined;
  }
  const schemaList = dbSchemaStore.getSchemaListByDatabaseId(atom.parentId);
  return schemaList.find((schema) => schema.name === atom.table!.schema);
});
const table = computed(() => {
  const atom = tableAtom.value;
  if (isUndefined(atom)) {
    return undefined;
  }
  return schema.value?.tables.find((table) => table.name === atom.table!.name);
});

const gotoAlterSchema = () => {
  if (isUndefined(tableAtom.value) || isUndefined(table.value)) {
    return;
  }

  const database = databaseStore.getDatabaseById(tableAtom.value.parentId);
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
    sql: `ALTER TABLE ${table.value.name}`,
  };
  const url = `/issue/new?${stringify(query)}`;
  window.open(url, "_blank");
};

const handleClosePane = () => {
  emit("close-pane");
};
</script>

<style scoped>
.table-schema {
  @apply h-full space-y-2;
}

.table-schema--header {
  @apply flex items-center p-2 border-b;
}

.table-schema--meta {
  @apply px-2 py-1;
  @apply border-b;
}

.table-schema--content {
  @apply px-2 py-1;
  height: calc(100% - 116px);
}
</style>
