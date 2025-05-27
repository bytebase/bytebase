<template>
  <div class="flex flex-col px-4 whitespace-nowrap text-sm">
    <div v-if="isDemo" class="flex gap-1 items-center text-accent py-2">
      <heroicons-outline:presentation-chart-bar class="w-5 h-5" />
      {{ $t("common.demo-mode") }}
    </div>
    <div v-else class="py-2">
      <div
        v-if="subscriptionViewMode === 'TRIAL'"
        class="flex items-center gap-1 text-accent cursor-pointer"
        @click="state.showTrialModal = true"
      >
        <heroicons-solid:sparkles class="w-5 h-5" />
        <span>{{ $t("subscription.plan.try") }}</span>
      </div>
      <router-link
        v-if="subscriptionViewMode === 'LINK'"
        :to="autoSubscriptionRoute($router)"
        exact-active-class=""
      >
        {{ $t(readableCurrentPlan) }}
      </router-link>
      <div v-if="subscriptionViewMode === 'PLAIN'">
        {{ $t(readableCurrentPlan) }}
      </div>
    </div>

    <NTooltip v-bind="tooltipProps">
      <template #trigger>
        <div
          class="flex items-center gap-1 py-2"
          :class="
            canUpgrade
              ? 'text-success cursor-pointer'
              : 'text-control-light cursor-default'
          "
          @click="state.showReleaseModal = canUpgrade"
        >
          <Volume2Icon v-if="canUpgrade" class="h-5 w-5" />
          {{ version }}
        </div>
      </template>
      <template #default>
        <div class="flex flex-col gap-y-1">
          <div v-if="canUpgrade" class="whitespace-nowrap">
            {{ $t("settings.release.new-version-available") }}
          </div>
          <div>BE Git hash: {{ gitCommitBE }}</div>
          <div>FE Git hash: {{ gitCommitFE }}</div>
        </div>
      </template>
    </NTooltip>
  </div>

  <TrialModal
    v-if="state.showTrialModal"
    @cancel="state.showTrialModal = false"
  />
  <ReleaseRemindModal
    v-if="!hideReleaseRemind && state.showReleaseModal"
    @cancel="state.showReleaseModal = false"
  />
</template>

<script lang="ts" setup>
import {
  useActuatorV1Store,
  useAppFeature,
  useSubscriptionV1Store,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { autoSubscriptionRoute, hasWorkspacePermissionV2 } from "@/utils";
import { Volume2Icon } from "lucide-vue-next";
import { NTooltip, type TooltipProps } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import ReleaseRemindModal from "../ReleaseRemindModal.vue";
import TrialModal from "../TrialModal.vue";

interface LocalState {
  showTrialModal: boolean;
  showReleaseModal: boolean;
}

defineProps<{
  tooltipProps?: TooltipProps;
}>();

const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();
const hideReleaseRemind = useAppFeature("bb.feature.hide-release-remind");
const hideTrial = useAppFeature("bb.feature.hide-trial");
const disallowNavigateToConsole = useAppFeature(
  "bb.feature.disallow-navigate-to-console"
);

const state = reactive<LocalState>({
  showTrialModal: false,
  showReleaseModal: false,
});

const { isDemo } = storeToRefs(actuatorStore);
const canUpgrade = computed(() => {
  return actuatorStore.hasNewRelease;
});

const subscriptionViewMode = computed(() => {
  if (disallowNavigateToConsole.value) {
    return "PLAIN";
  }
  if (
    !hideTrial.value &&
    subscriptionStore.currentPlan === PlanType.FREE &&
    hasWorkspacePermissionV2("bb.settings.set")
  ) {
    return "TRIAL";
  }
  return "LINK";
});

const readableCurrentPlan = computed((): string => {
  const plan = subscriptionStore.currentPlan;
  switch (plan) {
    case PlanType.TEAM:
      return "subscription.plan.team.title";
    case PlanType.ENTERPRISE:
      return "subscription.plan.enterprise.title";
    default:
      return "subscription.plan.free.title";
  }
});

const version = computed(() => {
  const v = actuatorStore.version;
  if (v.split(".").length == 3) {
    return `v${v}`;
  }
  return v;
});

const gitCommitBE = computed(() => {
  return `${actuatorStore.gitCommitBE.substring(0, 7)}`;
});
const gitCommitFE = computed(() => {
  return `${actuatorStore.gitCommitFE.substring(0, 7)}`;
});
</script>
