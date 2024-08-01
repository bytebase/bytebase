<template>
  <BBGrid
    :column-list="COLUMNS"
    :data-source="composedSlowQueryPolicyList"
    :row-clickable="false"
    :row-key="(item: ComposedSlowQueryPolicy) => item.instance.name"
    class="border"
  >
    <template #item="{ item }: { item: ComposedSlowQueryPolicy }">
      <div class="bb-grid-cell">
        <InstanceV1Name :instance="item.instance" :link="false" />
      </div>

      <div class="bb-grid-cell">
        <EnvironmentV1Name
          :environment="
            environmentStore.getEnvironmentByName(item.instance.environment)
          "
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
import {
  InstanceV1Name,
  EnvironmentV1Name,
  SpinnerSwitch,
} from "@/components/v2";
import { useCurrentUserV1, useEnvironmentV1Store } from "@/store";
import type { ComposedSlowQueryPolicy } from "@/types";
import type { InstanceResource } from "@/types/proto/v1/instance_service";
import { hasWorkspacePermissionV2 } from "@/utils";

defineProps<{
  composedSlowQueryPolicyList: ComposedSlowQueryPolicy[];
  toggleActive: (instance: InstanceResource, active: boolean) => Promise<void>;
}>();

const { t } = useI18n();
const environmentStore = useEnvironmentV1Store();
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
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update");
});
</script>
