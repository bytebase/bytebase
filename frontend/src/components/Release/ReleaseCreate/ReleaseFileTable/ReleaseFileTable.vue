<template>
  <div class="w-full flex flex-row items-center justify-between">
    <span class="textlabel !text-base">{{ $t("release.files") }}</span>
    <div>
      <NButton size="small" @click="onCreateFileClick">
        <template #icon>
          <PlusIcon />
        </template>
        {{ $t("release.actions.new-file") }}
      </NButton>
    </div>
  </div>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="sortedFiles"
    :row-props="rowProps"
    :striped="true"
    :row-key="(file) => file.version"
  />
  <div>
    <ul class="list-disc list-inside pl-2 text-sm leading-5 text-gray-500">
      <li>
        {{
          $t(
            "release.messages.files-will-always-be-sorted-and-executed-in-version-order"
          )
        }}
      </li>
      <li>
        {{ $t("release.messages.cannot-modify-files-after-created") }}
      </li>
    </ul>
  </div>

  <CreateReleaseFilesPanel
    :show="state.showCreatePanel"
    :file="state.selectedFile"
    @close="state.showCreatePanel = false"
  />
</template>

<script setup lang="tsx">
import { orderBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NDataTable, type DataTableColumn } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { Release_File } from "@/types/proto/v1/release_service";
import { useReleaseCreateContext, type FileToCreate } from "../context";
import CreateReleaseFilesPanel from "./CreateReleaseFilesPanel.vue";

interface LocalState {
  showCreatePanel: boolean;
  selectedFile?: FileToCreate;
}

const { t } = useI18n();
const { files } = useReleaseCreateContext();
const state = reactive<LocalState>({
  showCreatePanel: false,
});

const sortedFiles = computed(() => {
  return orderBy(files.value, "version", "desc");
});

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

const rowProps = (row: FileToCreate) => {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      state.showCreatePanel = true;
      state.selectedFile = row;
    },
  };
};

const onCreateFileClick = () => {
  state.showCreatePanel = true;
  state.selectedFile = undefined;
};
</script>
