<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="instanceRoleList"
    :striped="true"
    :bordered="true"
    v-bind="$attrs"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";

defineProps<{
  instanceRoleList: InstanceRole[];
}>();

const { t } = useI18n();
const columns = computed((): DataTableColumn<InstanceRole>[] => [
  {
    title: t("common.user"),
    key: "user",
    width: 200,
    render: (instanceRole) => instanceRole.roleName,
  },
  {
    title: t("instance.grants"),
    key: "grants",
    render: (instanceRole) => (
      <div class="whitespace-pre-wrap break-all">
        {(instanceRole.attribute ?? "").replaceAll("\n", "\n\n")}
      </div>
    ),
  },
]);
</script>
