<template>
  <div class="w-full space-y-4 text-sm">
    <FeatureAttention :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW" />

    <CustomApproval v-if="state.ready" />
    <div v-else class="w-full py-[4rem] flex justify-center items-center">
      <BBSpin />
    </div>
  </div>

  <ApprovalRuleDialog />

  <FeatureModal
    :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, toRef } from "vue";
import { BBSpin } from "@/bbkit";
import {
  CustomApproval,
  ApprovalRuleDialog,
  provideCustomApprovalContext,
  TabValueList,
} from "@/components/CustomApproval/Settings/components/CustomApproval/";
import { FeatureAttention, FeatureModal } from "@/components/FeatureGuard";
import { useRouteHash } from "@/composables/useRouteHash";
import {
  featureToRef,
  useWorkspaceApprovalSettingStore,
  useRiskStore,
} from "@/store";
import { PlanFeature } from "@/types/proto/v1/subscription_service";

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
const tab = useRouteHash("rules", TabValueList, "replace");
const hasCustomApprovalFeature = featureToRef(PlanFeature.FEATURE_APPROVAL_WORKFLOW);

provideCustomApprovalContext({
  hasFeature: hasCustomApprovalFeature,
  showFeatureModal: toRef(state, "showFeatureModal"),
  allowAdmin: toRef(props, "allowEdit"),
  ready: toRef(state, "ready"),
  tab,
  dialog: ref(),
});

onMounted(async () => {
  try {
    await Promise.all([
      useWorkspaceApprovalSettingStore().fetchConfig(),
      useRiskStore().fetchRiskList(),
    ]);
    state.ready = true;
  } catch {
    // nothing
  }
});
</script>
