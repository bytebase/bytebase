<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="data"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: ComposedDatabaseGroup) => data.name"
    :row-props="rowProps"
    :pagination="{ pageSize: 20 }"
    :paginate-single-page="false"
  />
</template>

<script lang="tsx" setup>
import {
  NButton,
  NDataTable,
  NTag,
  NDropdown,
  type DataTableColumn,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { ComposedDatabaseGroup } from "@/types";
import { hasPermissionToCreateChangeDatabaseIssueInProject } from "@/utils";
import { generateDatabaseGroupIssueRoute } from "@/utils/databaseGroup/issue";

type DatabaseGroupDataTableColumn = DataTableColumn<ComposedDatabaseGroup> & {
  hide?: boolean;
};

const props = withDefaults(
  defineProps<{
    databaseGroupList: ComposedDatabaseGroup[];
    bordered?: boolean;
    loading?: boolean;
    showProject?: boolean;
    customClick?: boolean;
  }>(),
  {
    bordered: true,
    showProject: true,
  }
);

const emit = defineEmits<{
  (
    event: "row-click",
    e: MouseEvent,
    databaseGroup: ComposedDatabaseGroup
  ): void;
  (event: "update:selected-database-groups", val: Set<string>): void;
  (event: "edit", databaseGroup: ComposedDatabaseGroup): void;
}>();

const { t } = useI18n();
const router = useRouter();

const columnList = computed((): DatabaseGroupDataTableColumn[] => {
  const NAME: DatabaseGroupDataTableColumn = {
    key: "title",
    title: t("common.name"),
    render: (data) => {
      return (
        <div class="space-x-2">
          <span>{data.databasePlaceholder}</span>
          {data.multitenancy && (
            <NTag round type="info" size="small">
              {t("database-group.multitenancy.self")}
            </NTag>
          )}
        </div>
      );
    },
  };
  const PROJECT: DatabaseGroupDataTableColumn = {
    key: "project",
    title: t("common.project"),
    hide: !props.showProject,
    render: (data) => {
      return <span>{data.projectEntity.title}</span>;
    },
  };
  const EDIT_BUTTON: DatabaseGroupDataTableColumn = {
    key: "actions",
    title: "",
    width: 150,
    render: (data) => {
      return (
        <div class="flex justify-end gap-2">
          <NDropdown
            trigger="hover"
            options={[
              {
                label: `${t("database.edit-schema")} (DDL)`,
                key: "bb.issue.database.schema.update",
              },
              {
                label: `${t("database.change-data")} (DML)`,
                key: "bb.issue.database.data.update",
              },
            ]}
            onSelect={(key) => doDatabaseGroupChangeAction(key, data)}
          >
            <NButton
              size="small"
              disabled={
                !hasPermissionToCreateChangeDatabaseIssueInProject(
                  data.projectEntity
                )
              }
            >
              {t("common.change")}
            </NButton>
          </NDropdown>
        </div>
      );
    },
  };

  // Maybe we can add more columns here. e.g. matched databases, etc.
  return [NAME, PROJECT, EDIT_BUTTON].filter((column) => !column.hide);
});

const data = computed(() => {
  return [...props.databaseGroupList];
});

const rowProps = (databaseGroup: ComposedDatabaseGroup) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      if (props.customClick) {
        emit("row-click", e, databaseGroup);
        return;
      }
    },
  };
};

const doDatabaseGroupChangeAction = (
  key: string,
  databaseGroup: ComposedDatabaseGroup
) => {
  const issueRoute = generateDatabaseGroupIssueRoute(key as any, databaseGroup);
  router.push(issueRoute);
};
</script>
