<template>
  <div class="w-full flex flex-row items-center justify-between">
    <span class="textlabel !text-base">{{ $t("release.files") }}</span>
    <div>
      <NDropdown
        trigger="hover"
        :options="addFileButtonOptions"
        @select="state.selectedNewFileOption = $event"
      >
        <NButton size="small">
          <template #icon>
            <PlusIcon />
          </template>
          {{ $t("release.actions.new-file") }}
        </NButton>
      </NDropdown>
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
    :show="state.selectedNewFileOption === 'raw-sql'"
    :file="state.selectedFile"
    @close="state.selectedNewFileOption = undefined"
  />

  <ImportFilesFromReleasePanel
    :show="state.selectedNewFileOption === 'import-from-release'"
    @close="state.selectedNewFileOption = undefined"
  />
</template>

<script setup lang="tsx">
import { orderBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import {
  NButton,
  NDropdown,
  NDataTable,
  type DataTableColumn,
  type DropdownOption,
} from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useReleaseCreateContext, type FileToCreate } from "../context";
import CreateReleaseFilesPanel from "./CreateReleaseFilesPanel.vue";
import ImportFilesFromReleasePanel from "./ImportFilesFromReleasePanel.vue";

interface LocalState {
  selectedNewFileOption?: "raw-sql" | "import-from-release";
  selectedFile?: FileToCreate;
}

const { t } = useI18n();
const { files } = useReleaseCreateContext();
const state = reactive<LocalState>({});

const sortedFiles = computed(() => {
  return orderBy(files.value, "version", "desc");
});

const columnList = computed(() => {
  const columns: DataTableColumn<FileToCreate>[] = [
    {
      key: "version",
      title: t("common.version"),
      width: 128,
      render: (file) => <span class="textlabel">{file.version}</span>,
    },
    {
      key: "title",
      title: t("database.revision.filename"),
      width: 160,
      ellipsis: true,
      render: (file) => {
        return file.path || "-";
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

const addFileButtonOptions = computed((): DropdownOption[] => {
  return [
    {
      label: t("schema-editor.raw-sql"),
      key: "raw-sql",
    },
    {
      label: t("release.actions.import-from-release"),
      key: "import-from-release",
    },
  ];
});

const rowProps = (row: FileToCreate) => {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      state.selectedNewFileOption = "raw-sql";
      state.selectedFile = row;
    },
  };
};
</script>
