<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="data"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: ComposedDatabaseGroup) => data.name"
    :checked-row-keys="Array.from(state.selectedDatabaseGroupNameList)"
    :row-props="rowProps"
    :pagination="{ pageSize: 20 }"
    :paginate-single-page="false"
    @update:checked-row-keys="
        (val) => (state.selectedDatabaseGroupNameList = new Set(val as string[]))
      "
  />
</template>

<script lang="tsx" setup>
import type { ComposedDatabaseGroup } from "@/types";
import { hasPermissionToCreateChangeDatabaseIssueInProject } from "@/utils";
import { generateDatabaseGroupIssueRoute } from "@/utils/databaseGroup/issue";
import {
  NButton,
  NDataTable,
  NDropdown,
  type DataTableColumn
} from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

interface LocalState {
  selectedDatabaseGroupNameList: Set<string>;
}

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
    showSelection?: boolean;
    showActions?: boolean;
    singleSelection?: boolean;
    selectedDatabaseGroupNames?: string[];
  }>(),
  {
    bordered: true,
    selectedDatabaseGroupNames: () => [],
  }
);

const emit = defineEmits<{
  (
    event: "row-click",
    e: MouseEvent,
    databaseGroup: ComposedDatabaseGroup
  ): void;
  (event: "update:selected-database-groups", val: Set<string>): void;
}>();

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  selectedDatabaseGroupNameList: new Set(props.selectedDatabaseGroupNames),
});

const columnList = computed((): DatabaseGroupDataTableColumn[] => {
  const SELECTION: DatabaseGroupDataTableColumn = {
    type: "selection",
    multiple: !props.singleSelection,
    hide: !props.showSelection,
    cellProps: () => {
      return {
        onClick: (e: MouseEvent) => {
          e.stopPropagation();
        },
      };
    },
  };
  const NAME: DatabaseGroupDataTableColumn = {
    key: "title",
    title: t("common.name"),
    render: (data) => {
      return (
        <div class="space-x-2">
          <span>{data.databasePlaceholder}</span>
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
  const ACTIONS: DatabaseGroupDataTableColumn = {
    key: "actions",
    title: "",
    width: 150,
    hide: !props.showActions,
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
  return [SELECTION, NAME, PROJECT, ACTIONS].filter((column) => !column.hide);
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

      if (props.singleSelection) {
        state.selectedDatabaseGroupNameList = new Set([databaseGroup.name]);
      } else {
        const selectedDatabaseGroupNameList = new Set(
          Array.from(state.selectedDatabaseGroupNameList)
        );
        if (selectedDatabaseGroupNameList.has(databaseGroup.name)) {
          selectedDatabaseGroupNameList.delete(databaseGroup.name);
        } else {
          selectedDatabaseGroupNameList.add(databaseGroup.name);
        }
        state.selectedDatabaseGroupNameList = selectedDatabaseGroupNameList;
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

watch(
  () => state.selectedDatabaseGroupNameList,
  () => {
    emit(
      "update:selected-database-groups",
      state.selectedDatabaseGroupNameList
    );
  }
);
</script>
