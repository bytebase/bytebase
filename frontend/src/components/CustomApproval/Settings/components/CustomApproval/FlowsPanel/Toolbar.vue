<template>
  <div class="flex items-center gap-x-2">
    <NButton type="primary" :disabled="!allowAdmin" @click="createRule">
      <FeatureBadge
        feature="bb.feature.custom-approval"
        custom-class="mr-1 text-white"
      />
      {{ $t("common.create") }}
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
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
