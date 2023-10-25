<template>
  <BBGrid
    :data-source="changes"
    :column-list="columns"
    :show-placeholder="true"
    :row-clickable="true"
    class="border"
    @click-row="handleClickRow"
  >
    <template #item="{ item: change, row }: BBGridRow<Change>">
      <div v-if="reorderMode" class="bb-grid-cell justify-center gap-x-1">
        <ReorderButtons
          v-if="reorderMode"
          :row="row"
          :changes="changes"
          @move="$emit('reorder-move', row, $event)"
        />
      </div>
      <div class="bb-grid-cell">
        <Source :change="change" />
      </div>

      <div class="bb-grid-cell">
        <DatabaseForChange :change="change" />
      </div>
      <div class="bb-grid-cell">
        <SQL :change="change" />
      </div>
      <div class="bb-grid-cell">
        <RemoveChangeButton @click="$emit('remove-change', change)" />
      </div>
    </template>
  </BBGrid>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridColumn, BBGridRow } from "@/bbkit";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
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

const columns = computed((): BBGridColumn[] => {
  const columns: BBGridColumn[] = [
    { title: t("changelist.change-source.source"), width: "auto" },
    { title: t("common.database"), width: "1fr" },
    { title: t("common.sql"), width: "3fr" },
    {
      title: "",
      width: "6rem",
    },
  ];
  if (props.reorderMode) {
    columns.unshift({
      title: "",
      width: "4rem",
      class: "justify-center",
    });
  }

  return columns;
});

const handleClickRow = (item: Change) => {
  emit("select-change", item);
};
</script>
