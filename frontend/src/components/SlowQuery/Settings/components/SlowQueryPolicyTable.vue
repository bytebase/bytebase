<template>
  <BBGrid
    :column-list="COLUMNS"
    :data-source="instanceList"
    :row-clickable="false"
    :show-placeholder="true"
    row-key="id"
    class="border w-[50rem] max-w-full"
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
          :on-toggle="(active) => handleToggle(instance, active)"
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
import { useSlowQueryPolicyStore } from "@/store";

const props = defineProps<{
  instanceList: Instance[];
  policyList: Policy[];
}>();

const { t } = useI18n();
const policyStore = useSlowQueryPolicyStore();

const COLUMNS = computed((): BBGridColumn[] => {
  return [
    {
      title: t("common.instance"),
      width: "minmax(auto, 18rem)",
    },
    {
      title: t("common.environment"),
      width: "minmax(auto, 10rem)",
    },
    {
      title: t("common.active"),
      width: "1fr",
    },
  ];
});

const isActive = (instance: Instance) => {
  const policy = props.policyList.find((policy) => {
    return policy.resourceId === instance.id;
  });
  if (!policy) return false;
  const payload = policy.payload as SlowQueryPolicyPayload;
  return payload.active;
};

const handleToggle = async (instance: Instance, active: boolean) => {
  try {
    const payload: SlowQueryPolicyPayload = {
      active,
    };
    await policyStore.upsertPolicyByResourceTypeAndPolicyType(
      "instance",
      instance.id,
      "bb.policy.slow-query",
      {
        payload,
      }
    );
  } catch {
    // nothing
  }
};
</script>
