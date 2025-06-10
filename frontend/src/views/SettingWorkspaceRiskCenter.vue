<template>
  <FeatureAttention :feature="PlanFeature.FEATURE_RISK_ASSESSMENT" class="mb-4" />

  <div class="w-full space-y-4 text-sm">
    <div class="textinfolabel">
      {{ $t("custom-approval.risk.description") }}
      <a
        href="https://www.bytebase.com/docs/administration/risk-center"
        target="_blank"
        class="normal-link inline-flex flex-row items-center"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>
    <RiskCenter v-if="state.ready" />
    <div v-else class="w-full py-[4rem] flex justify-center items-center">
      <BBSpin />
    </div>
  </div>

  <RiskDialog />

  <FeatureModal
    :feature="PlanFeature.FEATURE_RISK_ASSESSMENT"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { BBSpin } from "@/bbkit";
import {
  RiskCenter,
  RiskDialog,
  provideRiskCenterContext,
} from "@/components/CustomApproval/Settings/components/RiskCenter";
import { provideRiskFilter } from "@/components/CustomApproval/Settings/components/common";
import { FeatureAttention, FeatureModal } from "@/components/FeatureGuard";
import { featureToRef, useRiskStore } from "@/store";
import { PlanFeature } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { computed, onMounted, reactive, ref, toRef } from "vue";

interface LocalState {
  ready: boolean;
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  ready: false,
  showFeatureModal: false,
});
const hasRiskAssessmentFeature = featureToRef(PlanFeature.FEATURE_RISK_ASSESSMENT);

const allowAdmin = computed(() => {
  return hasWorkspacePermissionV2("bb.risks.update");
});

provideRiskFilter();
provideRiskCenterContext({
  hasFeature: hasRiskAssessmentFeature,
  showFeatureModal: toRef(state, "showFeatureModal"),
  allowAdmin,
  ready: toRef(state, "ready"),
  dialog: ref(),
});

onMounted(async () => {
  try {
    await useRiskStore().fetchRiskList();
    state.ready = true;
  } catch {
    // nothing
  }
});
</script>
