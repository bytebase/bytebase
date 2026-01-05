<template>
  <div class="w-full flex flex-col gap-y-6">
    <RulesSection
      v-for="source in supportedSourceList"
      :key="source"
      :source="source"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import RulesSection from "./RulesPanel/RulesSection.vue";

const supportedSourceList = computed(() => [
  WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE,
  WorkspaceApprovalSetting_Rule_Source.CREATE_DATABASE,
  WorkspaceApprovalSetting_Rule_Source.EXPORT_DATA,
  WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE,
  WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED,
]);
</script>
