<template>
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

  <div class="w-full mt-4 space-y-4 text-sm">
    <RiskCenter v-if="state.ready" />
    <div v-else class="w-full py-[4rem] flex justify-center items-center">
      <BBSpin />
    </div>
  </div>

  <RiskDialog />

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.custom-approval"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref, toRef } from "vue";

import { featureToRef, useCurrentUser, useRiskStore } from "@/store";
import { hasWorkspacePermission } from "@/utils";
import {
  RiskCenter,
  RiskDialog,
  provideRiskCenterContext,
} from "@/components/CustomApproval/Settings/components/RiskCenter";
import { provideRiskFilter } from "@/components/CustomApproval/Settings/components/common";

interface LocalState {
  ready: boolean;
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  ready: false,
  showFeatureModal: false,
});
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

const currentUser = useCurrentUser();
const allowAdmin = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-custom-approval",
    currentUser.value.role
  );
});

provideRiskFilter();
provideRiskCenterContext({
  hasFeature: hasCustomApprovalFeature,
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
