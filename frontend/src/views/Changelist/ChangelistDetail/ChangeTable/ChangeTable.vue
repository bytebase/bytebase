<template>
  <BBGrid
    :data-source="changes"
    :column-list="columns"
    :show-placeholder="true"
    :custom-header="true"
    :row-clickable="true"
    class="border"
    @click-row="handleClickRow"
  >
    <template #header>
      <div role="table-row" class="bb-grid-row bb-grid-header-row group">
        <div
          v-for="(column, index) in columns"
          :key="index"
          role="table-cell"
          class="bb-grid-header-cell capitalize"
          :class="[column.class]"
        >
          <template v-if="index === 0">
            <NCheckbox
              :class="[reorderMode && 'invisible']"
              :checked="allSelectionState.checked"
              :indeterminate="allSelectionState.indeterminate"
              @update:checked="toggleSelectAll"
            />
          </template>
          <template v-else>{{ column.title }}</template>
        </div>
      </div>
    </template>

    <template #item="{ item: change, row }: BBGridRow<Change>">
      <div class="bb-grid-cell justify-center gap-x-1">
        <NCheckbox
          v-if="!reorderMode"
          :checked="isSelected(change)"
          @update:checked="toggleSelect(change, $event)"
          @click.stop
        />
        <ReorderButtons
          v-else
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
import { NCheckbox } from "naive-ui";
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
  selected: Change[];
  reorderMode: boolean;
}>();

const emit = defineEmits<{
  (event: "update:selected", selected: Change[]): void;
  (event: "select-change", change: Change): void;
  (event: "remove-change", change: Change): void;
  (event: "reorder-move", row: number, delta: -1 | 1): void;
}>();

const { t } = useI18n();

const columns = computed((): BBGridColumn[] => {
  return [
    { title: "", width: "4rem", class: "justify-center" },
    { title: t("changelist.change-source.source"), width: "auto" },
    { title: t("common.database"), width: "1fr" },
    { title: t("common.sql"), width: "3fr" },
    {
      title: "",
      width: "6rem",
    },
  ];
});

const allSelectionState = computed(() => {
  const { changes, selected } = props;
  const set = new Set(selected);

  const checked =
    selected.length > 0 && changes.every((change) => set.has(change));
  const indeterminate = !checked && selected.some((name) => set.has(name));

  return {
    checked,
    indeterminate,
  };
});

const toggleSelectAll = (on: boolean) => {
  if (on) {
    emit("update:selected", [...props.changes]);
  } else {
    emit("update:selected", []);
  }
};

const isSelected = (change: Change) => {
  return props.selected.includes(change);
};

const toggleSelect = (change: Change, on: boolean) => {
  const set = new Set(props.selected);
  if (on) {
    if (!set.has(change)) {
      set.add(change);
      emit("update:selected", Array.from(set));
    }
  } else {
    if (set.has(change)) {
      set.delete(change);
      emit("update:selected", Array.from(set));
    }
  }
};

const handleClickRow = (item: Change) => {
  emit("select-change", item);
};
</script>
