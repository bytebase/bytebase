<template>
  <div class="w-full">
    <RulesPanel />
  </div>
</template>

<script lang="ts" setup>
import { provideRiskFilter } from "../common/RiskFilter";
import RulesPanel from "./RulesPanel";

provideRiskFilter();
</script>
