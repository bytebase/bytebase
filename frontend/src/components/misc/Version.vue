<template>
  <div class="flex flex-col px-4 whitespace-nowrap text-sm">
    <div v-if="isDemo" class="flex gap-1 items-center text-accent py-2">
      <heroicons-outline:presentation-chart-bar class="w-5 h-5" />
      {{ $t("common.demo-mode") }}
    </div>
    <div v-else class="py-2">
      <RequireEnterpriseButton
        v-if="subscriptionViewMode === 'TRIAL'"
        text
        size="small"
      >
        <template #icon>
          <SparklesIcon class="w-5 h-5 text-accent" />
        </template>
        {{ $t("subscription.plan.try") }}
      </RequireEnterpriseButton>
      <router-link
        v-if="subscriptionViewMode === 'LINK'"
        :to="autoSubscriptionRoute($router)"
        exact-active-class=""
      >
        {{ $t(readableCurrentPlan) }}
      </router-link>
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
            {{ $t("remind.release.new-version-available") }}
          </div>
          <div>BE Git hash: {{ gitCommitBE }}</div>
          <div>FE Git hash: {{ gitCommitFE }}</div>
        </div>
      </template>
    </NTooltip>
  </div>

  <ReleaseRemindModal
    v-if="state.showReleaseModal"
    @cancel="state.showReleaseModal = false"
  />
</template>

<script lang="ts" setup>
import { SparklesIcon, Volume2Icon } from "lucide-vue-next";
import { NTooltip, type TooltipProps } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import RequireEnterpriseButton from "@/components/RequireEnterpriseButton.vue";
import {
  useActuatorV1Store,
  useAppFeature,
  useSubscriptionV1Store,
} from "@/store";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { autoSubscriptionRoute, hasWorkspacePermissionV2 } from "@/utils";
import ReleaseRemindModal from "../ReleaseRemindModal.vue";

interface LocalState {
  showReleaseModal: boolean;
}

defineProps<{
  tooltipProps?: TooltipProps;
}>();

const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();
const hideTrial = useAppFeature("bb.feature.hide-trial");

const state = reactive<LocalState>({
  showReleaseModal: false,
});

const { isDemo } = storeToRefs(actuatorStore);
const canUpgrade = computed(() => {
  return actuatorStore.hasNewRelease;
});

const subscriptionViewMode = computed(() => {
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
