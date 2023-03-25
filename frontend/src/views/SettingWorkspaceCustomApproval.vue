<template>
  <div class="w-full mt-4 space-y-4 text-sm">
    <CustomApproval v-if="state.ready" />
    <div v-else class="w-full py-[4rem] flex justify-center items-center">
      <BBSpin />
    </div>
  </div>

  <ApprovalRuleDialog />

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.custom-approval"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref, toRef } from "vue";

import {
  featureToRef,
  useWorkspaceApprovalSettingStore,
  useCurrentUser,
} from "@/store";
import { hasWorkspacePermission } from "@/utils";
import {
  CustomApproval,
  ApprovalRuleDialog,
  provideCustomApprovalContext,
  TabValueList,
} from "@/components/CustomApproval/Settings/components/CustomApproval/";
import { useRouteHash } from "@/composables/useRouteHash";

interface LocalState {
  ready: boolean;
  showFeatureModal: boolean;
}

const store = useWorkspaceApprovalSettingStore();
const state = reactive<LocalState>({
  ready: false,
  showFeatureModal: false,
});
const tab = useRouteHash("rules", TabValueList, "replace");
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

const currentUser = useCurrentUser();
const allowAdmin = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-custom-approval",
    currentUser.value.role
  );
});

provideCustomApprovalContext({
  hasFeature: hasCustomApprovalFeature,
  showFeatureModal: toRef(state, "showFeatureModal"),
  allowAdmin,
  ready: toRef(state, "ready"),
  tab,
  dialog: ref(),
});

onMounted(async () => {
  try {
    await Promise.all([store.fetchConfig()]);
    state.ready = true;
  } catch {
    // nothing
  }
});
</script>
