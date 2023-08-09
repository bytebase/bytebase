<template>
  <BBGrid
    :column-list="columnList"
    :data-source="instanceRoleList"
    :row-clickable="false"
  >
    <template #item="{ item: instanceRole }: { item: InstanceRole }">
      <div class="bb-grid-cell">
        {{ instanceRole.roleName }}
      </div>
      <div class="bb-grid-cell whitespace-pre-wrap break-all">
        {{ (instanceRole.attribute ?? "").replaceAll("\n", "\n\n") }}
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridColumn } from "@/bbkit";
import { InstanceRole } from "@/types/proto/v1/instance_role_service";

defineProps<{
  instanceRoleList: InstanceRole[];
}>();

const { t } = useI18n();
const columnList = computed((): BBGridColumn[] => [
  {
    title: t("common.User"),
    width: "minmax(auto, 12rem)",
  },
  {
    title: t("instance.grants"),
    width: "1fr",
  },
]);
</script>
