<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="databaseResourceList"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="tsx" setup>
import { NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { watch } from "vue";
import { useI18n } from "vue-i18n";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";
import { useDatabaseV1Store, batchGetOrFetchDatabases } from "@/store";
import type { DatabaseResource } from "@/types";
import type { DatabaseGroup } from "@/types/proto/v1/database_group_service";

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
    databaseResource.databaseFullName
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
    databaseResource.databaseFullName
  );
  return database;
};

watch(
  () => props.databaseResourceList,
  async () => {
    await batchGetOrFetchDatabases(
      props.databaseResourceList.map(
        (databaseResource) => databaseResource.databaseFullName
      )
    );
  },
  {
    immediate: true,
  }
);

const columns = computed((): DataTableColumn<DatabaseResource>[] => {
  return [
    {
      title: t("common.database"),
      key: "database",
      render: (row) => extractDatabaseName(row),
    },
    {
      title: t("common.table"),
      key: "table",
      render: (row) => (
        <span class="line-clamp-1">{extractTableName(row)}</span>
      ),
    },
    {
      title: t("common.environment"),
      key: "environment",
      render: (row) => (
        <EnvironmentV1Name
          environment={extractComposedDatabase(row).effectiveEnvironmentEntity}
        />
      ),
    },
    {
      title: t("common.instance"),
      key: "instance",
      render: (row) => (
        <InstanceV1Name
          instance={extractComposedDatabase(row).instanceResource}
        />
      ),
    },
  ];
});
</script>
