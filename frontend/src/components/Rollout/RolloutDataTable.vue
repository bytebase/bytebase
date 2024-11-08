<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="sortedRolloutList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(rollout: ComposedRollout) => rollout.name"
    :row-props="rowProps"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAvatar } from "@/bbkit";
import { getTimeForPbTimestamp, type ComposedRollout } from "@/types";
import { humanizeTs } from "@/utils";

const props = withDefaults(
  defineProps<{
    rolloutList: ComposedRollout[];
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

const columnList = computed(
  (): (DataTableColumn<ComposedRollout> & { hide?: boolean })[] => {
    const columns: (DataTableColumn<ComposedRollout> & { hide?: boolean })[] = [
      {
        key: "title",
        width: 160,
        title: t("common.title"),
        render: (rollout) => {
          return (
            <p class="inline-flex w-full">
              <span class="shrink truncate">{rollout.title}</span>
            </p>
          );
        },
      },
      {
        key: "creator",
        title: t("common.creator"),
        width: 128,
        render: (rollout) => (
          <div class="flex flex-row items-center overflow-hidden gap-x-2">
            <BBAvatar size="SMALL" username={rollout.creatorEntity.title} />
            <span class="truncate">{rollout.creatorEntity.title}</span>
          </div>
        ),
      },
      {
        key: "createTime",
        title: t("common.created-at"),
        width: 128,
        render: (rollout) =>
          humanizeTs(getTimeForPbTimestamp(rollout.createTime, 0) / 1000),
      },
    ];
    return columns.filter((column) => !column.hide);
  }
);

const sortedRolloutList = computed(() => {
  return props.rolloutList;
});

const rowProps = (rollout: ComposedRollout) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const url = `/${rollout.name}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};
</script>
