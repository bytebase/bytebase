<template>
  <NDataTable
    :columns="columns"
    :data="dbExtensionList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="ts" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import type { PropType } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ExtensionMetadata } from "@/types/proto/v1/database_service";

defineProps({
  dbExtensionList: {
    required: true,
    type: Object as PropType<ExtensionMetadata[]>,
  },
});

const { t } = useI18n();

const columns = computed(() => {
  return [
    {
      key: "name",
      title: t("common.name"),
      render: (row) => {
        return row.name;
      },
    },
    {
      key: "version",
      title: t("common.version"),
      render: (row) => {
        return row.version;
      },
    },
    {
      key: "schema",
      title: t("common.schema"),
      render: (row) => {
        return row.schema;
      },
    },
    {
      key: "description",
      title: t("common.description"),
      render: (row) => {
        return row.description;
      },
    },
  ] as DataTableColumn<ExtensionMetadata>[];
});
</script>
