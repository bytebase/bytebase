<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="migrationFiles"
    :striped="true"
    :row-key="(file) => file.version"
  />
</template>

<script setup lang="tsx">
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { Release_File } from "@/types/proto/v1/release_service";
import { useReleaseDetailContext } from "../context";

const { release } = useReleaseDetailContext();

const migrationFiles = computed(() => release.value.files);

const columnList = computed(() => {
  const columns: DataTableColumn<Release_File>[] = [
    {
      key: "version",
      title: "Version",
      ellipsis: true,
      render: (file) => file.version,
    },
    {
      key: "title",
      title: "Filename",
      ellipsis: true,
      render: (file) => file.name,
    },
    {
      key: "sheetSha256",
      width: 150,
      title: "Hash",
      render: (file) => file.sheetSha256,
    },
    {
      key: "actions",
      width: 150,
      title: "Actions",
      render: (file) => file.sheetSha256,
    },
  ];
  return columns;
});
</script>
