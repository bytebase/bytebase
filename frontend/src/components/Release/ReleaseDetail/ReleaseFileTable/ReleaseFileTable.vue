<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="release.files"
    :row-props="rowProps"
    :striped="true"
    :row-key="(file) => file.version"
  />

  <Drawer
    :show="!!state.selectedReleaseFile"
    @close="state.selectedReleaseFile = undefined"
  >
    <DrawerContent
      style="width: 75vw; max-width: calc(100vw - 8rem)"
      :title="'Release File'"
    >
      <ReleaseFileDetailPanel
        v-if="state.selectedReleaseFile"
        :release="release"
        :release-file="state.selectedReleaseFile"
      />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="tsx">
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed, reactive } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { Release_File } from "@/types/proto/v1/release_service";
import { useReleaseDetailContext } from "../context";
import ReleaseFileDetailPanel from "./ReleaseFileDetailPanel.vue";

interface LocalState {
  selectedReleaseFile?: Release_File;
}

const { release } = useReleaseDetailContext();
const state = reactive<LocalState>({});

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

const rowProps = (row: Release_File) => {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      state.selectedReleaseFile = row;
    },
  };
};
</script>
