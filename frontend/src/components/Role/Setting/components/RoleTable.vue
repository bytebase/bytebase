<template>
  <BBGrid
    :column-list="COLUMNS"
    :data-source="roleList"
    :row-clickable="false"
    row-key="id"
    class="border"
  >
    <template #item="{ item: role }: RoleGridRow">
      <RoleTableRow :role="role" @edit="$emit('select-role', $event)" />
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import { Role } from "@/types/proto/v1/role_service";
import RoleTableRow from "./RoleTableRow.vue";

export type RoleGridRow = BBGridRow<Role>;

defineProps<{
  roleList: Role[];
}>();

defineEmits<{
  (event: "select-role", role: Role): void;
}>();

const { t } = useI18n();

const COLUMNS = computed((): BBGridColumn[] => {
  return [
    {
      title: t("role.title"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("common.description"),
      width: "3fr",
    },
    {
      title: t("common.operation"),
      width: "8rem",
    },
  ];
});
</script>
