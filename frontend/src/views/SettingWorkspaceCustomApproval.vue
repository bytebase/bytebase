<template>
  <div class="w-full flex flex-col gap-y-4 text-sm">
    <FeatureAttention :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW" />
    <BBAttention v-if="hasCustomApprovalFeature" type="info" :description="$t('custom-approval.rule.first-match-wins')" />

    <CustomApproval v-if="state.ready" />
    <div v-else class="w-full py-16 flex justify-center items-center">
      <BBSpin />
    </div>
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { onMounted, reactive, toRef } from "vue";
import { BBAttention, BBSpin } from "@/bbkit";
import {
  CustomApproval,
  provideCustomApprovalContext,
} from "@/components/CustomApproval/Settings/components/CustomApproval/";
import { FeatureAttention, FeatureModal } from "@/components/FeatureGuard";
import { featureToRef, useWorkspaceApprovalSettingStore } from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

interface LocalState {
  ready: boolean;
  showFeatureModal: boolean;
}

const props = defineProps<{
  allowEdit: boolean;
}>();

const state = reactive<LocalState>({
  ready: false,
  showFeatureModal: false,
});
const hasCustomApprovalFeature = featureToRef(
  PlanFeature.FEATURE_APPROVAL_WORKFLOW
);

provideCustomApprovalContext({
  hasFeature: hasCustomApprovalFeature,
  showFeatureModal: toRef(state, "showFeatureModal"),
  allowAdmin: toRef(props, "allowEdit"),
  ready: toRef(state, "ready"),
});

onMounted(async () => {
  try {
    await useWorkspaceApprovalSettingStore().fetchConfig();
    state.ready = true;
  } catch {
    // nothing
  }
});
</script>
