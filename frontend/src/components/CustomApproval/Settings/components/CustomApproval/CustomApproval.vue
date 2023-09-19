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
      <NTabPane
        name="external-approval"
        :tab="$t('custom-approval.approval-flow.external-approval.self')"
        display-directive="show:lazy"
      >
        <ExternalApprovalNodesPanel />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { provideRiskFilter } from "../common/RiskFilter";
import ExternalApprovalNodesPanel from "./ExternalApprovalNodesPanel";
import FlowsPanel from "./FlowsPanel";
import RulesPanel from "./RulesPanel";
import Toolbar from "./Toolbar.vue";
import { useCustomApprovalContext } from "./context";

const { tab } = useCustomApprovalContext();
provideRiskFilter();
</script>
