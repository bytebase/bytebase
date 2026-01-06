<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="files"
    :row-props="rowProps"
    :striped="true"
    :row-key="(file) => file.path"
    :checked-row-keys="selectedFileIds"
    @update:checked-row-keys="(val) => onRowSelected(val as string[])"
  />
</template>

<script setup lang="tsx">
import { type DataTableColumn, NDataTable } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  type Release_File,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";

const props = withDefaults(
  defineProps<{
    files: Release_File[];
    releaseType: Release_Type;
    showSelection?: boolean;
    rowClickable?: boolean;
    selectedFiles?: Release_File[];
  }>(),
  {
    showSelection: false,
    rowClickable: true,
    selectedFiles: () => [],
  }
);

const emit = defineEmits<{
  (event: "row-click", e: MouseEvent, val: Release_File): void;
  (event: "update:selected-files", val: Release_File[]): void;
}>();

const { t } = useI18n();

// Track currently selected file paths, initialize with prop values
const selectedFileIds = ref<string[]>(props.selectedFiles.map((f) => f.path));

const columnList = computed(() => {
  const columns: (DataTableColumn<Release_File> & {
    hide?: boolean;
  })[] = [
    {
      type: "selection",
      hide: !props.showSelection,
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
    },
    {
      key: "version",
      title: t("common.version"),
      width: 160,
      ellipsis: true,
      className: "textlabel",
      render: (file) => file.version,
    },
    {
      key: "type",
      title: t("common.type"),
      width: 64,
      resizable: true,
      render: (file) => getReleaseFileTypeText(file),
    },
    {
      key: "filename",
      title: t("database.revision.filename"),
      width: 128,
      ellipsis: true,
      render: (file) => file.path || "-",
    },
  ];
  return columns.filter((column) => !column.hide);
});

const getReleaseFileTypeText = (file: Release_File) => {
  switch (props.releaseType) {
    case Release_Type.DECLARATIVE:
      return "SDL";
    case Release_Type.VERSIONED:
      return file.enableGhost
        ? `${t("issue.title.change-database")} (gh-ost)`
        : t("issue.title.change-database");
    case Release_Type.TYPE_UNSPECIFIED:
      return "";
    default:
      props.releaseType satisfies never;
      return "";
  }
};

const rowProps = (file: Release_File) => {
  return {
    style: props.rowClickable || props.showSelection ? "cursor: pointer;" : "",
    onClick: (e: MouseEvent) => {
      // Check if we're in selection mode
      if (props.showSelection) {
        // Don't toggle if clicking on the checkbox itself
        const target = e.target as HTMLElement;
        if (target.closest(".n-checkbox")) {
          return;
        }

        // Trigger selection through NDataTable's mechanism
        // We need to maintain the current selection and toggle this file
        const currentSelectedIds = selectedFileIds.value;
        const filePath = file.path;

        let newSelectedIds: string[];
        if (currentSelectedIds.includes(filePath)) {
          // Deselect
          newSelectedIds = currentSelectedIds.filter((id) => id !== filePath);
        } else {
          // Select
          newSelectedIds = [...currentSelectedIds, filePath];
        }

        onRowSelected(newSelectedIds);
      } else if (props.rowClickable) {
        // Emit row-click event for other purposes
        emit("row-click", e, file);
      }
    },
  };
};

const onRowSelected = (val: string[]) => {
  selectedFileIds.value = val;
  emit(
    "update:selected-files",
    val.map((path) => props.files.find((f) => f.path === path)!)
  );
};

// Watch for external selection changes
watch(
  () => props.selectedFiles,
  (newFiles) => {
    selectedFileIds.value = newFiles.map((f) => f.path);
  }
);
</script>
