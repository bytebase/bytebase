<template>
  <div class="w-full mt-4 space-y-4 text-sm">
    <FeatureAttentionForInstanceLicense
      v-if="hasCustomApprovalFeature"
      feature="bb.feature.custom-approval"
    />
    <FeatureAttention v-else feature="bb.feature.custom-approval" />

    <CustomApproval v-if="state.ready" />
    <div v-else class="w-full py-[4rem] flex justify-center items-center">
      <BBSpin />
    </div>
  </div>

  <ApprovalRuleDialog />
  <ExternalApprovalNodeDrawer />

  <FeatureModal
    feature="bb.feature.custom-approval"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref, toRef } from "vue";

import {
  featureToRef,
  useWorkspaceApprovalSettingStore,
  useCurrentUserV1,
  useRiskStore,
} from "@/store";
import { hasWorkspacePermissionV1 } from "@/utils";
import {
  CustomApproval,
  ApprovalRuleDialog,
  ExternalApprovalNodeDrawer,
  provideCustomApprovalContext,
  TabValueList,
} from "@/components/CustomApproval/Settings/components/CustomApproval/";
import { useRouteHash } from "@/composables/useRouteHash";

interface LocalState {
  ready: boolean;
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  ready: false,
  showFeatureModal: false,
});
const tab = useRouteHash("rules", TabValueList, "replace");
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

const currentUserV1 = useCurrentUserV1();
const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-custom-approval",
    currentUserV1.value.userRole
  );
});

provideCustomApprovalContext({
  hasFeature: hasCustomApprovalFeature,
  showFeatureModal: toRef(state, "showFeatureModal"),
  allowAdmin,
  ready: toRef(state, "ready"),
  tab,
  dialog: ref(),
  externalApprovalNodeContext: ref(),
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
