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
import { I18nT, useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { useUserStore } from "@/store";
import { getDateForPbTimestamp } from "@/types";
import type { Changelist } from "@/types/proto/v1/changelist_service";
import {
  extractUserResourceName,
  extractChangelistResourceName,
} from "@/utils";
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
        key: "updated-by",
        title: t("common.updated-at"),
        width: 256,
        render: (changelist) => {
          return (
            <I18nT keypath="common.updated-at-by">
              {{
                time: () => (
                  <HumanizeDate
                    date={getDateForPbTimestamp(changelist.createTime)}
                  />
                ),
                user: () => getUser(changelist.creator)?.title,
              }}
            </I18nT>
          );
        },
      },
      {
        key: "created-by",
        title: t("common.created-at"),
        width: 256,
        render: (changelist) => {
          return (
            <I18nT keypath="common.created-at-by">
              {{
                time: () => (
                  <HumanizeDate
                    date={getDateForPbTimestamp(changelist.createTime)}
                  />
                ),
                user: () => getUser(changelist.creator)?.title,
              }}
            </I18nT>
          );
        },
      },
    ] as ChangelistDataTableColumn[]
  ).filter((column) => !column.hide);
});

const getUser = (name: string) => {
  const email = extractUserResourceName(name);
  return useUserStore().getUserByEmail(email);
};

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
