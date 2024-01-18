<template>
  <div class="w-full space-y-4 text-sm">
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
import { onMounted, reactive, ref, toRef } from "vue";
import {
  CustomApproval,
  ApprovalRuleDialog,
  ExternalApprovalNodeDrawer,
  provideCustomApprovalContext,
  TabValueList,
} from "@/components/CustomApproval/Settings/components/CustomApproval/";
import { useRouteHash } from "@/composables/useRouteHash";
import {
  featureToRef,
  useWorkspaceApprovalSettingStore,
  useRiskStore,
} from "@/store";

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
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

provideCustomApprovalContext({
  hasFeature: hasCustomApprovalFeature,
  showFeatureModal: toRef(state, "showFeatureModal"),
  allowAdmin: toRef(props, "allowEdit"),
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
