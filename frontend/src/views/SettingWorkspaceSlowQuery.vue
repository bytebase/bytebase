<template>
  <div class="w-full mt-4 space-y-4 text-sm">
    <FeatureAttention
      v-if="!hasSlowQueryFeature"
      feature="bb.feature.slow-query"
      :description="$t('subscription.features.bb-feature-slow-query.desc')"
    />

    <SlowQuerySettings v-if="state.ready" />
    <div v-else class="w-full py-[4rem] flex justify-center items-center">
      <BBSpin />
    </div>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.slow-query"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";

import { featureToRef, useCurrentUser } from "@/store";
import { hasWorkspacePermission } from "@/utils";
import { SlowQuerySettings } from "@/components/SlowQuery";

interface LocalState {
  ready: boolean;
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  ready: false,
  showFeatureModal: false,
});
const hasSlowQueryFeature = featureToRef("bb.feature.slow-query");

const currentUser = useCurrentUser();
const allowAdmin = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-slow-query",
    currentUser.value.role
  );
});

setTimeout(() => {
  state.ready = true;
}, 500);
</script>
