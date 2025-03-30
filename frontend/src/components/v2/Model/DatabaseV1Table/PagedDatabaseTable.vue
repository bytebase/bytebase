<template>
  <PagedTable
    ref="databasePagedTable"
    :session-key="`bb.databases-table.${parent}`"
    :fetch-list="fetchDatabses"
    :class="customClass"
    :footer-class="footerClass"
  >
    <template #table="{ list, loading }">
      <DatabaseV1Table
        v-bind="$attrs"
        :key="`database-table.${parent}`"
        :loading="loading"
        :database-list="list"
        :keyword="filter.query"
        :custom-click="true"
        @row-click="handleDatabaseClick"
        @update:selected-databases="$emit('update:selected-databases', $event)"
      />
    </template>
  </PagedTable>
</template>

<script setup lang="tsx">
import { ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useRouter } from "vue-router";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useDatabaseV1Store, type DatabaseFilter } from "@/store";
import type { ComposedDatabase } from "@/types";
import { autoDatabaseRoute } from "@/utils";
import DatabaseV1Table from "./DatabaseV1Table.vue";

const props = withDefaults(
  defineProps<{
    filter?: DatabaseFilter;
    parent: string;
    customClass?: string;
    footerClass?: string;
    customClick?: boolean;
  }>(),
  {
    filter: () => ({}),
    customClass: "",
    footerClass: "",
    customClick: false,
  }
);

const emit = defineEmits<{
  (event: "row-click", e: MouseEvent, val: ComposedDatabase): void;
  (event: "update:selected-databases", val: Set<string>): void;
}>();

const databaseStore = useDatabaseV1Store();
const router = useRouter();

const databasePagedTable =
  ref<ComponentExposed<typeof PagedTable<ComposedDatabase>>>();

const fetchDatabses = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, databases } = await databaseStore.fetchDatabases({
    pageToken,
    pageSize,
    parent: props.parent,
    filter: props.filter,
  });
  return {
    nextPageToken,
    list: databases,
  };
};

watch(
  () => [props.filter, props.parent],
  () => databasePagedTable.value?.refresh(),
  { deep: true }
);

const handleDatabaseClick = (event: MouseEvent, database: ComposedDatabase) => {
  if (props.customClick) {
    emit("row-click", event, database);
  } else {
    const url = router.resolve(autoDatabaseRoute(router, database)).fullPath;
    if (event.ctrlKey || event.metaKey) {
      window.open(url, "_blank");
    } else {
      router.push(url);
    }
  }
};

defineExpose({
  updateCache: (databases: ComposedDatabase[]) => {
    databasePagedTable.value?.updateCache(databases);
  },
  refresh: () => databasePagedTable.value?.refresh(),
});
</script>
