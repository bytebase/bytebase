<template>
  <div class="w-full">
    <NTabs v-model:value="tab">
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
import { NTabPane, NTabs } from "naive-ui";
import { provideRiskFilter } from "../common/RiskFilter";
import { useCustomApprovalContext } from "./context";
import FlowsPanel from "./FlowsPanel";
import RulesPanel from "./RulesPanel";

const { tab } = useCustomApprovalContext();
provideRiskFilter();
</script>
