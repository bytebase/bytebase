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
import { NDataTable, NTag, type DataTableColumn } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import { Release_File } from "@/types/proto/v1/release_service";
import { useReleaseDetailContext } from "../context";
import ReleaseFileDetailPanel from "./ReleaseFileDetailPanel.vue";

interface LocalState {
  selectedReleaseFile?: Release_File;
}

const { t } = useI18n();
const { release } = useReleaseDetailContext();
const state = reactive<LocalState>({});

const columnList = computed(() => {
  const columns: DataTableColumn<Release_File>[] = [
    {
      key: "version",
      title: t("common.version"),
      width: 128,
      render: (file) => <span class="textlabel">{file.version}</span>,
    },
    {
      key: "title",
      title: t("database.revision.filename"),
      width: 256,
      ellipsis: true,
      render: (file) => {
        return (
          <div class="space-x-2">
            <span>{file.name}</span>
            <NTag
              v-if="schemaVersion"
              class="text-sm font-mono"
              size="small"
              round
            >
              {file.sheetSha256.slice(0, 8)}
            </NTag>
          </div>
        );
      },
    },
    {
      key: "statement",
      title: t("common.statement"),
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
