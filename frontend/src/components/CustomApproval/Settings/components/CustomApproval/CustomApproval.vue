<template>
  <div class="w-full">
    <NTabs v-model:value="tab">
      <template #suffix>
        <Toolbar />
      </template>
      <NTabPane
        name="rules"
        :tab="$t('custom-approval.rule.rules')"
        display-directive="show:lazy"
      >
        <RulesPanel />
      </NTabPane>
      <NTabPane
        name="flows"
        :tab="$t('custom-approval.approval-flow.approval-flows')"
        display-directive="show:lazy"
      >
        <FlowsPanel />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { useCustomApprovalContext } from "./context";
import RulesPanel from "./RulesPanel";
import FlowsPanel from "./FlowsPanel";
import Toolbar from "./Toolbar.vue";
import { provideRiskFilter } from "../common/RiskFilter";

const { tab } = useCustomApprovalContext();
provideRiskFilter();
</script>
