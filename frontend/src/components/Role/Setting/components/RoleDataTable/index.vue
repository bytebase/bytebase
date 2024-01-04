<template>
  <NDataTable
    :columns="columns"
    :data="roleList"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="ts" setup>
import { DataTableColumn, NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { Role } from "@/types/proto/v1/role_service";
import { displayRoleDescription } from "@/utils";
import RoleOperationsCell from "./cells/RoleOperationsCell.vue";
import RoleTitleCell from "./cells/RoleTitleCell.vue";
import RoleTypeCell from "./cells/RoleTypeCell.vue";

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
      key: "type",
      title: t("common.type"),
      width: "4rem",
      render: (role: Role) => {
        return h(RoleTypeCell, {
          role,
        });
      },
    },
    {
      key: "title",
      title: t("role.title"),
      width: "16rem",
      render: (role: Role) => {
        return h(RoleTitleCell, {
          role,
        });
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
        return h(RoleOperationsCell, {
          role,
          onEdit: () => {
            emit("select-role", role);
          },
        });
      },
    },
  ] as DataTableColumn<Role>[];
});
</script>
