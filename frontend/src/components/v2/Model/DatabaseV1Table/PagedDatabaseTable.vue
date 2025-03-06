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
        :keyword="keyword"
        :custom-click="true"
        @row-click="handleDatabaseClick"
        @update:selected-databases="$emit('update:selected-databases', $event)"
      />
    </template>
  </PagedTable>
</template>

<script setup lang="tsx">
import { useDebounceFn } from "@vueuse/core";
import { ref, watch, computed } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useRouter } from "vue-router";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useDatabaseV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { autoDatabaseRoute } from "@/utils";
import DatabaseV1Table from "./DatabaseV1Table.vue";

interface DatabaseFilter {
  project?: string;
  instance?: string;
  environment?: string;
  query?: string;
  showDeleted?: boolean;
  excludeUnassigned?: boolean;
  // label should be "{label key}:{label value}" format
  labels?: string[];
  engines?: Engine[];
  excludeEngines?: Engine[];
}

const props = withDefaults(
  defineProps<{
    filter?: DatabaseFilter;
    parent: string;
    customClass?: string;
    footerClass?: string;
    customClick?: boolean;
  }>(),
  {
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

const keyword = computed(() => {
  return props.filter?.query?.trim()?.toLowerCase();
});

const filter = computed(() => {
  const params: string[] = [];
  if (props.filter?.project) {
    params.push(`project == "${props.filter?.project}"`);
  }
  if (props.filter?.instance) {
    params.push(`instance == "${props.filter?.instance}"`);
  }
  if (props.filter?.environment) {
    params.push(`environment == "${props.filter?.environment}"`);
  }
  if (props.filter?.excludeUnassigned) {
    params.push(`exclude_unassigned == true`);
  }
  if (props.filter?.engines) {
    // engine filter should be:
    // engine in ["MYSQL", "POSTGRES"]
    params.push(
      `engine in [${props.filter?.engines.map((e) => `"${e}"`).join(", ")}]`
    );
  } else if (props.filter?.excludeEngines) {
    // engine filter should be:
    // !(engine in ["REDIS", "MONGODB"])
    params.push(
      `!(engine in [${props.filter?.excludeEngines.map((e) => `"${e}"`).join(", ")}])`
    );
  }
  if (keyword.value) {
    params.push(`name.matches("${keyword.value}")`);
  }
  if (props.filter?.labels) {
    // label filter like:
    // label == "region:asia,europe" && label == "tenant:bytebase"
    const labelMap = new Map<string, Set<string>>();
    for (const label of props.filter.labels) {
      const sections = label.split(":");
      if (sections.length !== 2) {
        continue;
      }
      if (!labelMap.has(sections[0])) {
        labelMap.set(sections[0], new Set());
      }
      labelMap.get(sections[0])!.add(sections[1]);
    }
    for (const [labelKey, labelValues] of labelMap.entries()) {
      params.push(`label == "${labelKey}:${[...labelValues].join(",")}"`);
    }
  }

  return params.join(" && ");
});

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
    filter: filter.value,
    parent: props.parent,
    showDeleted: props.filter?.showDeleted,
  });
  return {
    nextPageToken,
    list: databases,
  };
};

watch(
  () => [filter.value, props.parent],
  useDebounceFn(async () => {
    await databasePagedTable.value?.refresh();
  }, 500)
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
</script>
