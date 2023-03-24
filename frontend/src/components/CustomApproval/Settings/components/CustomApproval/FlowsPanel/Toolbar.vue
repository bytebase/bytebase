<template>
  <div class="flex items-center gap-x-2">
    <NButton type="primary" :disabled="!allowAdmin" @click="createRule">
      {{ $t("common.create") }}
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";

import { useCustomApprovalContext } from "../context";
import { emptyLocalApprovalRule } from "../logic";
const context = useCustomApprovalContext();
const { allowAdmin, dialog } = context;

const createRule = () => {
  dialog.value = {
    mode: "CREATE",
    rule: emptyLocalApprovalRule(),
  };
};
</script>
