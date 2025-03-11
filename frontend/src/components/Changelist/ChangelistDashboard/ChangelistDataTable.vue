<template>
  <NDataTable
    key="change-list-table"
    size="small"
    :columns="columnList"
    :data="changelists"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: Changelist) => data.name"
    :row-props="rowProps"
    :pagination="pagination"
    :paginate-single-page="false"
  />
</template>

<script setup lang="tsx">
import { NDataTable } from "naive-ui";
import type { DataTableColumn, PaginationProps } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAvatar } from "@/bbkit";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { useUserStore } from "@/store";
import { getDateForPbTimestamp } from "@/types";
import type { Changelist } from "@/types/proto/v1/changelist_service";
import { extractChangelistResourceName } from "@/utils";
import { projectForChangelist } from "../ChangelistDetail/common";

type ChangelistDataTableColumn = DataTableColumn<Changelist> & {
  hide?: boolean;
};

const props = withDefaults(
  defineProps<{
    changelists: Changelist[];
    bordered?: boolean;
    loading?: boolean;
    pagination?: false | PaginationProps;
    showProject: boolean;
  }>(),
  {
    bordered: true,
    showProject: true,
    pagination: () => ({ pageSize: 20 }) as PaginationProps,
  }
);

const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();

const columnList = computed((): ChangelistDataTableColumn[] => {
  return (
    [
      {
        key: "name",
        title: t("changelist.self"),
        render: (changelist) => {
          return changelist.description;
        },
      },
      {
        key: "project",
        title: t("common.project"),
        width: 256,
        hide: !props.showProject,
        render: (changelist) => {
          return projectForChangelist(changelist).title;
        },
      },
      {
        key: "updated-at",
        title: t("common.updated-at"),
        width: 256,
        render: (changelist) => {
          return (
            <HumanizeDate date={getDateForPbTimestamp(changelist.updateTime)} />
          );
        },
      },
      {
        key: "creator",
        title: t("common.creator"),
        width: 256,
        render: (changelist) => {
          const creator = userStore.getUserByIdentifier(changelist.creator);
          if (!creator) return null;
          return (
            <div class="flex flex-row items-center overflow-hidden gap-x-1">
              <BBAvatar size="SMALL" username={creator.title} />
              <span class="truncate">{creator.title}</span>
            </div>
          );
        },
      },
    ] as ChangelistDataTableColumn[]
  ).filter((column) => !column.hide);
});

const rowProps = (changelist: Changelist) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const route = router.resolve({
        path: `changelists/${extractChangelistResourceName(changelist.name)}`,
      });
      if (e.ctrlKey || e.metaKey) {
        window.open(route.fullPath, "_blank");
      } else {
        router.push(route);
      }
    },
  };
};
</script>
