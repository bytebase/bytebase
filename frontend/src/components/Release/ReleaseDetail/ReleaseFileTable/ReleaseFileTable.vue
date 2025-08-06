<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="files"
    :row-props="rowProps"
    :striped="true"
    :row-key="(file) => file.id"
    :checked-row-keys="selectedFileIds"
    @update:checked-row-keys="(val) => onRowSelected(val as string[])"
  />
</template>

<script setup lang="tsx">
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { Release_File } from "@/types/proto-es/v1/release_service_pb";
import { bytesToString, getReleaseFileStatement } from "@/utils";

const props = withDefaults(
  defineProps<{
    files: Release_File[];
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

// Track currently selected file IDs, initialize with prop values
const selectedFileIds = ref<string[]>(props.selectedFiles.map((f) => f.id));

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
      key: "filename",
      title: t("database.revision.filename"),
      width: 128,
      ellipsis: true,
      render: (file) => file.path || "-",
    },
    {
      key: "statement-size",
      title: t("common.statement-size"),
      width: 128,
      ellipsis: true,
      render: (file) => bytesToString(Number(file.statementSize)),
    },
    {
      key: "statement",
      title: t("common.statement"),
      ellipsis: true,
      render: (file) => getReleaseFileStatement(file),
    },
  ];
  return columns.filter((column) => !column.hide);
});

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
        const fileId = file.id;

        let newSelectedIds: string[];
        if (currentSelectedIds.includes(fileId)) {
          // Deselect
          newSelectedIds = currentSelectedIds.filter((id) => id !== fileId);
        } else {
          // Select
          newSelectedIds = [...currentSelectedIds, fileId];
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
    val.map((id) => props.files.find((f) => f.id === id)!)
  );
};

// Watch for external selection changes
watch(
  () => props.selectedFiles,
  (newFiles) => {
    selectedFileIds.value = newFiles.map((f) => f.id);
  }
);
</script>
