<template>
  <BBGrid
    :column-list="columns"
    :ready="!isFetching"
    :data-source="branchList"
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
          class="bb-grid-header-cell capitalize"
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
    <template #item="{ item: branch }: BBGridRow<Branch>">
      <div class="bb-grid-cell">
        <NCheckbox
          :checked="isSelected(branch)"
          @update:checked="toggleSelect(branch, $event)"
          @click.stop
        />
      </div>

      <!-- eslint-disable-next-line vue/no-v-html -->
      <div class="bb-grid-cell" v-html="renderTitle(branch)"></div>
      <div class="bb-grid-cell">
        <BranchBaseline :branch="branch" />
      </div>
      <div class="bb-grid-cell">
        <i18n-t keypath="common.updated-at-by">
          <template #time>
            <HumanizeDate :date="branch.updateTime" />
          </template>
          <template #user>{{ getUser(branch.updater)?.title }}</template>
        </i18n-t>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { escape } from "lodash-es";
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridColumn, BBGridRow } from "@/bbkit";
import BranchBaseline from "@/components/Branch/BranchBaseline.vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { useUserStore } from "@/store";
import { Branch } from "@/types/proto/v1/branch_service";
import { extractUserResourceName, getHighlightHTMLByRegExp } from "@/utils";

const props = defineProps<{
  selected: string[];
  branchList: Branch[];
  isFetching: boolean;
  keyword: string;
}>();

const emit = defineEmits<{
  (event: "update:selected", selected: string[]): void;
  (event: "click-item", branch: Branch): void;
}>();

const { t } = useI18n();

const columns = computed((): BBGridColumn[] => {
  return [
    { title: "", width: "auto" },
    { title: t("common.branch"), width: "1fr" },
    { title: t("schema-designer.baseline-version"), width: "2fr" },
    {
      title: t("common.updated-at"),
      width: "auto",
    },
  ];
});

const allSelectionState = computed(() => {
  const { branchList: list, selected } = props;
  const set = new Set(selected);

  const checked =
    selected.length > 0 && list.every((branch) => set.has(branch.name));
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
      props.branchList.map((branch) => branch.name)
    );
  } else {
    emit("update:selected", []);
  }
};

const isSelected = (branch: Branch) => {
  return props.selected.includes(branch.name);
};

const toggleSelect = (branch: Branch, on: boolean) => {
  const set = new Set(props.selected);
  const key = branch.name;
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

const renderTitle = (item: Branch) => {
  const keyword = props.keyword.trim().toLowerCase();

  const { title } = item;

  if (!keyword) {
    return escape(title);
  }

  return getHighlightHTMLByRegExp(
    escape(title),
    escape(keyword),
    false /* !caseSensitive */
  );
};

const getUser = (name: string) => {
  const email = extractUserResourceName(name);
  return useUserStore().getUserByEmail(email);
};

const handleClickRow = (item: Branch) => {
  emit("click-item", item);
};
</script>
