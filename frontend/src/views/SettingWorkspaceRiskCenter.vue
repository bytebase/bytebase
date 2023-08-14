<template>
  <FeatureAttentionForInstanceLicense
    v-if="hasCustomApprovalFeature"
    feature="bb.feature.custom-approval"
    custom-class="my-4"
  />
  <FeatureAttention
    v-else
    feature="bb.feature.custom-approval"
    custom-class="my-4"
  />

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
    feature="bb.feature.custom-approval"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref, toRef } from "vue";
import {
  RiskCenter,
  RiskDialog,
  provideRiskCenterContext,
} from "@/components/CustomApproval/Settings/components/RiskCenter";
import { provideRiskFilter } from "@/components/CustomApproval/Settings/components/common";
import { featureToRef, useCurrentUserV1, useRiskStore } from "@/store";
import { hasWorkspacePermissionV1 } from "@/utils";

interface LocalState {
  ready: boolean;
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  ready: false,
  showFeatureModal: false,
});
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

const currentUserV1 = useCurrentUserV1();
const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-custom-approval",
    currentUserV1.value.userRole
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
