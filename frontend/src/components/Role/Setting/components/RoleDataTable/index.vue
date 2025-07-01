<template>
  <NDataTable
    key="custom-role-table"
    :columns="columns"
    :data="roleList"
    :striped="true"
    :row-key="(role: Role) => role.name"
    :bordered="true"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import { displayRoleDescription } from "@/utils";
import RoleOperationsCell from "./cells/RoleOperationsCell.vue";
import RoleTitleCell from "./cells/RoleTitleCell.vue";

defineProps<{
  roleList: Role[];
}>();

const emit = defineEmits<{
  (event: "select-role", role: Role): void;
}>();

const { t } = useI18n();

const columns = computed(() => {
  return [
    {
      key: "title",
      title: t("role.title"),
      width: "16rem",
      render: (role: Role) => {
        return <RoleTitleCell role={role} />;
      },
    },
    {
      key: "description",
      title: t("common.description"),
      ellipsis: {
        tooltip: true,
      },
      render: (role) => {
        return displayRoleDescription(role.name);
      },
    },
    {
      key: "operations",
      title: "",
      width: "10rem",
      render: (role: Role) => {
        return (
          <RoleOperationsCell
            role={role}
            onEdit={() => emit("select-role", role)}
          />
        );
      },
    },
  ] as DataTableColumn<Role>[];
});
</script>
