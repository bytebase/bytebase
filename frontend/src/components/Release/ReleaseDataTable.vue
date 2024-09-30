<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="sortedReleaseList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(release: ComposedRelease) => release.name"
    :checked-row-keys="Array.from(state.selectedReleaseNameList)"
    :row-props="rowProps"
    @update:checked-row-keys="
        (val) => (state.selectedReleaseNameList = new Set(val as string[]))
      "
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAvatar } from "@/bbkit";
import { getTimeForPbTimestamp, type ComposedRelease } from "@/types";
import { humanizeTs } from "@/utils";

interface LocalState {
  selectedReleaseNameList: Set<string>;
}

const props = withDefaults(
  defineProps<{
    releaseList: ComposedRelease[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
  }>(),
  {
    loading: true,
    bordered: false,
    showSelection: false,
  }
);

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  selectedReleaseNameList: new Set(),
});

const columnList = computed((): DataTableColumn<ComposedRelease>[] => {
  const columns: (DataTableColumn<ComposedRelease> & { hide?: boolean })[] = [
    {
      type: "selection",
      width: 40,
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
      hide: !props.showSelection,
    },
    {
      key: "title",
      title: t("common.title"),
      ellipsis: true,
      render: (release) => {
        return <div class="flex items-center space-x-2">{release.title}</div>;
      },
    },
    {
      key: "createTime",
      title: t("common.created-at"),
      width: 150,
      render: (release) =>
        humanizeTs(getTimeForPbTimestamp(release.createTime, 0) / 1000),
    },
    {
      key: "creator",
      title: t("common.creator"),
      width: 150,
      render: (release) => (
        <div class="flex flex-row items-center overflow-hidden gap-x-2">
          <BBAvatar size="SMALL" username={release.creatorEntity.title} />
          <span class="truncate">{release.creatorEntity.title}</span>
        </div>
      ),
    },
  ];
  return columns.filter((column) => !column.hide);
});

const sortedReleaseList = computed(() => {
  return props.releaseList;
});

const rowProps = (release: ComposedRelease) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const url = `/${release.name}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};

watch(
  () => props.releaseList,
  (list) => {
    const oldReleaseNames = Array.from(state.selectedReleaseNameList.values());
    const newReleaseNames = new Set(list.map((release) => release.name));
    oldReleaseNames.forEach((name) => {
      if (!newReleaseNames.has(name)) {
        state.selectedReleaseNameList.delete(name);
      }
    });
  }
);
</script>
