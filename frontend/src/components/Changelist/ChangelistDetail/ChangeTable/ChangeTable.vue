<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="changes"
    :striped="true"
    :bordered="true"
    :row-props="getRowProps"
  />
</template>

<script setup lang="tsx">
import { NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import DatabaseForChange from "./DatabaseForChange.vue";
import RemoveChangeButton from "./RemoveChangeButton.vue";
import ReorderButtons from "./ReorderButtons.vue";
import SQL from "./SQL.vue";
import Source from "./Source.vue";

const props = defineProps<{
  changes: Change[];
  reorderMode: boolean;
}>();

const emit = defineEmits<{
  (event: "select-change", change: Change): void;
  (event: "remove-change", change: Change): void;
  (event: "reorder-move", row: number, delta: -1 | 1): void;
}>();

const { t } = useI18n();

const columns = computed((): DataTableColumn<Change>[] => {
  const cols: DataTableColumn<Change>[] = [];

  if (props.reorderMode) {
    cols.push({
      title: "",
      key: "reorder",
      width: 64,
      align: "center",
      render: (change, index) => (
        <div class="flex justify-center gap-x-1">
          <ReorderButtons
            row={index}
            changes={props.changes}
            onMove={(delta: -1 | 1) => emit("reorder-move", index, delta)}
          />
        </div>
      ),
    });
  }

  cols.push(
    {
      title: t("changelist.change-source.source"),
      key: "source",
      width: 120,
      render: (change) => <Source change={change} />,
    },
    {
      title: t("common.database"),
      key: "database",
      render: (change) => <DatabaseForChange change={change} />,
    },
    {
      title: t("common.sql"),
      key: "sql",
      render: (change) => <SQL change={change} />,
    },
    {
      title: "",
      key: "operations",
      width: 96,
      render: (change) => (
        <RemoveChangeButton onClick={() => emit("remove-change", change)} />
      ),
    }
  );

  return cols;
});

const getRowProps = (change: Change) => {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      emit("select-change", change);
    },
  };
};
</script>
