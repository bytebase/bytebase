<!-- This component is used when uses don't have available plan to access the feature, or the instance is missing required license. -->
<!-- Normally this should be a blocker to use the feature -->
<template>
  <div
    v-if="instanceMissingLicense"
    class="text-accent cursor-pointer"
    v-bind="$attrs"
    @click="
      (e: MouseEvent) => {
        e.stopPropagation();
        e.preventDefault();
        state.showInstanceAssignmentDrawer = true;
      }
    "
  >
    <NTooltip :show-arrow="true">
      <template #trigger>
        <heroicons-solid:lock-closed class="w-5 h-5" />
      </template>
      <span class="w-56 text-sm">
        {{ $t("subscription.instance-assignment.missing-license-attention") }}
      </span>
    </NTooltip>
  </div>
  <div v-else-if="!hasFeature" class="text-accent" v-bind="$attrs">
    <NTooltip :show-arrow="true">
      <template #trigger>
        <router-link
          v-if="clickable"
          :to="autoSubscriptionRoute($router)"
          exact-active-class=""
        >
          <SparklesIcon class="w-5 h-5" />
        </router-link>
        <span v-else>
          <SparklesIcon class="w-5 h-5" />
        </span>
      </template>
      <span class="w-56 text-sm">
        {{
          $t("subscription.require-subscription", {
            requiredPlan: $t(
              `subscription.plan.${PlanType[
                subscriptionStore.getMinimumRequiredPlan(feature)
              ].toLowerCase()}.title`
            ),
          })
        }}
      </span>
    </NTooltip>
  </div>
  <InstanceAssignment
    v-if="instanceMissingLicense && canManageSubscription"
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { SparklesIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { useSubscriptionV1Store } from "@/store";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { autoSubscriptionRoute, hasWorkspacePermissionV2 } from "@/utils";
import InstanceAssignment from "../InstanceAssignment.vue";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const props = withDefaults(
  defineProps<{
    feature: PlanFeature;
    instance?: Instance | InstanceResource;
    showInstanceMissingLicense?: boolean;
    clickable?: boolean;
  }>(),
  {
    instance: undefined,
    clickable: false,
    showInstanceMissingLicense: true,
  }
);

const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
});

const subscriptionStore = useSubscriptionV1Store();

const hasFeature = computed(() => {
  return subscriptionStore.hasInstanceFeature(props.feature);
});

const instanceMissingLicense = computed(() => {
  return (
    props.showInstanceMissingLicense &&
    subscriptionStore.instanceMissingLicense(props.feature, props.instance)
  );
});

const canManageSubscription = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.instances.update");
});
</script>
