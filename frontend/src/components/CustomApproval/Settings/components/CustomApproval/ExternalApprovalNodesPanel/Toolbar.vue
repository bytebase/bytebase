<template>
  <div class="flex items-center gap-x-2">
    <NButton type="primary" :disabled="!allowAdmin" @click="createNode">
      {{ $t("common.create") }}
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { ExternalApprovalSetting_Node } from "@/types/proto/v1/setting_service";
import { useCustomApprovalContext } from "../context";

const context = useCustomApprovalContext();
const {
  hasFeature,
  showFeatureModal,
  allowAdmin,
  externalApprovalNodeContext,
} = context;

const createNode = () => {
  if (!hasFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  externalApprovalNodeContext.value = {
    mode: "CREATE",
    node: ExternalApprovalSetting_Node.fromJSON({
      id: uuidv4(),
    }),
  };
};
</script>
