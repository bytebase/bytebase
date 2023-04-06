<template>
  <BBGrid
    :column-list="COLUMNS"
    :data-source="instanceList"
    :row-clickable="false"
    row-key="id"
    class="border"
  >
    <template #item="{ item: instance }: { item: Instance }">
      <div class="bb-grid-cell">
        <InstanceName :instance="instance" />
      </div>

      <div class="bb-grid-cell">
        <EnvironmentName :environment="instance.environment" />
      </div>
      <div class="bb-grid-cell">
        <SpinnerSwitch
          :value="isActive(instance)"
          :disabled="!allowAdmin"
          :on-toggle="(active) => toggleActive(instance, active)"
        />
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { type BBGridColumn, BBGrid } from "@/bbkit";
import type { Instance, Policy, SlowQueryPolicyPayload } from "@/types";
import { InstanceName, EnvironmentName, SpinnerSwitch } from "@/components/v2/";
import { useCurrentUser } from "@/store";
import { hasWorkspacePermission } from "@/utils";

const props = defineProps<{
  instanceList: Instance[];
  policyList: Policy[];
  toggleActive: (instance: Instance, active: boolean) => Promise<void>;
}>();

const { t } = useI18n();
const currentUser = useCurrentUser();

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
      title: t("common.active"),
      width: "minmax(auto, 6rem)",
    },
  ];
});

const allowAdmin = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-slow-query",
    currentUser.value.role
  );
});

const isActive = (instance: Instance) => {
  const policy = props.policyList.find((policy) => {
    return policy.resourceId === instance.id;
  });
  if (!policy) return false;
  const payload = policy.payload as SlowQueryPolicyPayload;
  return payload.active;
};
</script>
