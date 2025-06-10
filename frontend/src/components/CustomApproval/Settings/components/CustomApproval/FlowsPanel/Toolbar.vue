<template>
  <div class="flex items-center gap-x-2">
    <NButton type="primary" :disabled="!allowAdmin" @click="createRule">
      <FeatureBadge
        :feature="PlanLimitConfig_Feature.APPROVAL_WORKFLOW"
        class="mr-1 text-white"
      />
      {{ $t("common.create") }}
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { FeatureBadge } from "@/components/FeatureGuard";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";
import { useCustomApprovalContext } from "../context";
import { emptyLocalApprovalRule } from "../logic";

const context = useCustomApprovalContext();
const { hasFeature, showFeatureModal, allowAdmin, dialog } = context;

const createRule = () => {
  if (!hasFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  dialog.value = {
    mode: "CREATE",
    rule: emptyLocalApprovalRule(),
  };
};
</script>
