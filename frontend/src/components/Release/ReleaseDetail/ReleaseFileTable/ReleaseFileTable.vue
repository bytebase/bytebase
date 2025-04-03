<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="files"
    :row-props="rowProps"
    :striped="true"
    :row-key="(file) => file.id"
    @update:checked-row-keys="(val) => onRowSelected(val as string[])"
  />
</template>

<script setup lang="tsx">
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Release_File } from "@/types/proto/v1/release_service";
import { bytesToString, getReleaseFileStatement } from "@/utils";

const props = withDefaults(
  defineProps<{
    files: Release_File[];
    showSelection?: boolean;
    rowClickable?: boolean;
  }>(),
  {
    showSelection: false,
    rowClickable: true,
  }
);

const emit = defineEmits<{
  (event: "row-click", e: MouseEvent, val: Release_File): void;
  (event: "update:selected-files", val: Release_File[]): void;
}>();

const { t } = useI18n();

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
      render: (file) => bytesToString(file.statementSize.toNumber()),
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
    style: props.rowClickable ? "cursor: pointer;" : "",
    onClick: (e: MouseEvent) => {
      if (!props.rowClickable) {
        return;
      }

      emit("row-click", e, file);
    },
  };
};

const onRowSelected = (val: string[]) => {
  emit(
    "update:selected-files",
    val.map((id) => props.files.find((f) => f.id === id)!)
  );
};
</script>
