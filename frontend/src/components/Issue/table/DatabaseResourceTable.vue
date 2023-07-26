<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="databaseResourceList"
    :row-clickable="true"
    row-key="name"
    class="border"
  >
    <template #item="{ item }">
      <div class="bb-grid-cell">
        {{ extractDatabaseName(item) }}
      </div>
      <div class="bb-grid-cell">
        <span class="line-clamp-1">{{ extractTableName(item) }}</span>
      </div>
      <div class="bb-grid-cell">
        <EnvironmentV1Name
          :environment="
            extractComposedDatabase(item).instanceEntity.environmentEntity
          "
        />
      </div>
      <div class="bb-grid-cell truncate">
        <InstanceV1Name
          :instance="extractComposedDatabase(item).instanceEntity"
        />
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn } from "@/bbkit";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { DatabaseResource } from "@/types";
import { watch } from "vue";
import { useDatabaseV1Store } from "@/store";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";

const props = defineProps<{
  databaseResourceList: DatabaseResource[];
}>();

defineEmits<{
  (event: "edit", databaseGroup: DatabaseGroup): void;
}>();

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();

const extractDatabaseName = (databaseResource: DatabaseResource) => {
  const database = databaseStore.getDatabaseByName(
    String(databaseResource.databaseName)
  );
  return database.databaseName;
};

const extractTableName = (databaseResource: DatabaseResource) => {
  if (!databaseResource.schema && !databaseResource.table) {
    return "*";
  }
  const names = [];
  if (databaseResource.schema) {
    names.push(databaseResource.schema);
  }
  names.push(databaseResource.table || "*");
  return names.join(".");
};

const extractComposedDatabase = (databaseResource: DatabaseResource) => {
  const database = databaseStore.getDatabaseByName(
    String(databaseResource.databaseName)
  );
  return database;
};

watch(
  () => props.databaseResourceList,
  async () => {
    for (const databaseResource of props.databaseResourceList) {
      await databaseStore.getOrFetchDatabaseByName(
        databaseResource.databaseName
      );
    }
  },
  {
    immediate: true,
  }
);

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: t("common.database"), width: "1fr" },
    { title: t("common.table"), width: "0.5fr" },
    {
      title: t("common.environment"),
      width: "0.5fr",
    },
    { title: t("common.instance"), width: "1fr" },
  ];

  return columns;
});
</script>
