<template>
  <div v-if="tableInfo" class="table-schema">
    <div class="table-schema--header">
      <div class="table-schema--header-title mr-1 flex items-center">
        <heroicons-outline:table class="h-4 w-4 mr-1" />
        <span class="font-semibold">{{ tableInfo.name }}</span>
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
        <span>{{ tableInfo.rowCount }} rows</span>
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
        v-for="(column, index) in tableInfo.columnList"
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
import { ref, watch } from "vue";
import { useRouter } from "vue-router";

import { Database, DatabaseId, UNKNOWN_ID } from "@/types";
import { useSQLEditorStore, useTableStore } from "@/store";

const emit = defineEmits<{
  (e: "close-pane"): void;
}>();

const tableStore = useTableStore();
const sqlEditorStore = useSQLEditorStore();

const tableInfo = ref();
const ctx = sqlEditorStore.connectionContext;
const router = useRouter();

const gotoAlterSchema = () => {
  let databaseId = ctx.databaseId as DatabaseId;
  if (databaseId === UNKNOWN_ID) {
    const option = ctx.option;
    databaseId = option.parentId as number;
  }

  const projectId = sqlEditorStore.findProjectIdByDatabaseId(databaseId);
  const databaseList =
    sqlEditorStore.connectionInfo.databaseListByProjectId.get(projectId) as any;
  const databaseName = databaseList.find(
    (database: Database) => database.id === databaseId
  ).name;
  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.database.schema.update",
      name: `[${databaseName}] Alter schema`,
      project: projectId,
      databaseList: databaseId,
      sql: `ALTER TABLE ${ctx.tableName}`,
    },
  });
};

const handleClosePane = () => {
  emit("close-pane");
};

watch(
  () => sqlEditorStore.connectionContext.option,
  async (option) => {
    if (option && option.type === "table") {
      const res = await tableStore.fetchTableByDatabaseIdAndTableName({
        databaseId: option.parentId as number,
        tableName: option.label as string,
      });

      tableInfo.value = res;
    } else {
      tableInfo.value = null;
    }
  },
  { deep: true }
);
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
