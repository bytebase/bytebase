<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="release.files"
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

const columnList = computed(() => {
  const columns: DataTableColumn<Release_File>[] = [
    {
      key: "version",
      title: "Version",
      width: 150,
      render: (file) => file.version,
    },
    {
      key: "title",
      title: "Filename",
      width: 200,
      ellipsis: true,
      render: (file) => file.name,
    },
    {
      key: "sheetSha256",
      title: "Hash",
      width: 150,
      render: (file) => {
        return <code class={"text-sm"}>{file.sheetSha256.slice(0, 8)}</code>;
      },
    },
    {
      key: "statement",
      title: "Statement",
      ellipsis: true,
      render: (file) => file.statement,
    },
  ];
  return columns;
});
</script>
