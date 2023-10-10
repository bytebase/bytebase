<template>
  <BBGrid
    :column-list="columns"
    :ready="!isFetching"
    :data-source="changeHistoryList"
    :show-placeholder="true"
    :custom-header="true"
    class="border"
    @click-row="handleClickRow"
  >
    <template #header>
      <div role="table-row" class="bb-grid-row bb-grid-header-row group">
        <div
          v-for="(column, index) in columns"
          :key="index"
          role="table-cell"
          class="bb-grid-header-cell capitalize whitespace-nowrap"
          :class="[column.class]"
        >
          <template v-if="index === 0">
            <NCheckbox
              :checked="allSelectionState.checked"
              :indeterminate="allSelectionState.indeterminate"
              @update:checked="toggleSelectAll"
            />
          </template>
          <template v-else>{{ column.title }}</template>
        </div>
      </div>
    </template>
    <template #item="{ item: changeHistory }: BBGridRow<ChangeHistory>">
      <div class="bb-grid-cell">
        <NCheckbox
          :checked="isSelected(changeHistory)"
          @update:checked="toggleSelect(changeHistory, $event)"
          @click.stop
        />
      </div>

      <div class="bb-grid-cell">
        {{ displaySemanticType(changeHistory.type) }}
      </div>
      <div class="bb-grid-cell">
        <!-- eslint-disable-next-line vue/no-v-html -->
        <span class="whitespace-nowrap" v-html="renderVersion(changeHistory)" />
      </div>
      <div class="bb-grid-cell whitespace-nowrap">
        <IssueUID :change-history="changeHistory" />
      </div>
      <div class="bb-grid-cell">
        <Tables :change-history="changeHistory" />
      </div>
      <div class="bb-grid-cell overflow-hidden">
        <SQL :change-history="changeHistory" />
      </div>
    </template>
  </BBGrid>
</template>

<script setup lang="ts">
import { escape } from "lodash-es";
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridRow, BBGridColumn } from "@/bbkit";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { getHighlightHTMLByRegExp } from "@/utils";
import { displaySemanticType } from "../utils";
import IssueUID from "./IssueUID.vue";
import SQL from "./SQL.vue";
import Tables from "./Tables.vue";

const props = defineProps<{
  selected: string[];
  changeHistoryList: ChangeHistory[];
  isFetching: boolean;
  keyword: string;
}>();

const emit = defineEmits<{
  (event: "update:selected", selected: string[]): void;
  (event: "click-item", change: ChangeHistory): void;
}>();

const { t } = useI18n();

const columns = computed((): BBGridColumn[] => {
  return [
    { title: "", width: "auto" },
    { title: t("common.type"), width: "auto" },
    { title: t("common.version"), width: "1fr" },
    { title: t("common.issue"), width: "auto" },
    {
      title: t("changelist.change-source.change-history.tables"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("common.sql"),
      width: "3fr",
    },
  ];
});

const allSelectionState = computed(() => {
  const { changeHistoryList: list, selected } = props;
  const set = new Set(selected);

  const checked =
    selected.length > 0 &&
    list.every((changeHistory) => set.has(changeHistory.name));
  const indeterminate = !checked && selected.some((name) => set.has(name));

  return {
    checked,
    indeterminate,
  };
});

const toggleSelectAll = (on: boolean) => {
  if (on) {
    emit(
      "update:selected",
      props.changeHistoryList.map((changeHistory) => changeHistory.name)
    );
  } else {
    emit("update:selected", []);
  }
};

const isSelected = (changeHistory: ChangeHistory) => {
  return props.selected.includes(changeHistory.name);
};

const toggleSelect = (changeHistory: ChangeHistory, on: boolean) => {
  const set = new Set(props.selected);
  const key = changeHistory.name;
  if (on) {
    if (!set.has(key)) {
      set.add(key);
      emit("update:selected", Array.from(set));
    }
  } else {
    if (set.has(key)) {
      set.delete(key);
      emit("update:selected", Array.from(set));
    }
  }
};

const renderVersion = (item: ChangeHistory) => {
  const keyword = props.keyword.trim().toLowerCase();

  const { version } = item;

  if (!keyword) {
    return escape(version);
  }

  return getHighlightHTMLByRegExp(
    escape(version),
    escape(keyword),
    false /* !caseSensitive */
  );
};

const handleClickRow = (item: ChangeHistory) => {
  emit("click-item", item);
};
</script>
