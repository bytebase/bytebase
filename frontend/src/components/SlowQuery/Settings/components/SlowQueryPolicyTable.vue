<template>
  <BBGrid
    :column-list="COLUMNS"
    :data-source="composedSlowQueryPolicyList"
    :row-clickable="false"
    :row-key="(item: ComposedSlowQueryPolicy) => item.instance.id"
    class="border"
  >
    <template #item="{ item }: { item: ComposedSlowQueryPolicy }">
      <div class="bb-grid-cell">
        <InstanceName :instance="item.instance" :link="false" />
      </div>

      <div class="bb-grid-cell">
        <EnvironmentName
          :environment="item.instance.environment"
          :link="false"
        />
      </div>
      <div class="bb-grid-cell">
        <SpinnerSwitch
          :value="item.active"
          :disabled="!allowAdmin"
          :on-toggle="(active) => toggleActive(item.instance, active)"
        />
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { type BBGridColumn, BBGrid } from "@/bbkit";
import type { Instance, ComposedSlowQueryPolicy } from "@/types";
import { InstanceName, EnvironmentName, SpinnerSwitch } from "@/components/v2/";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV1 } from "@/utils";

defineProps<{
  composedSlowQueryPolicyList: ComposedSlowQueryPolicy[];
  toggleActive: (instance: Instance, active: boolean) => Promise<void>;
}>();

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();

const COLUMNS = computed((): BBGridColumn[] => {
  return [
    {
      title: t("common.instance"),
      width: "2fr",
    },
    {
      title: t("common.environment"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("slow-query.report"),
      width: "minmax(auto, 6rem)",
    },
  ];
});

const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-slow-query",
    currentUserV1.value.userRole
  );
});
</script>
