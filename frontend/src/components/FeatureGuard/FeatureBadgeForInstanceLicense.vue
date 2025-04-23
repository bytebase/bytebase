<!-- This component is used when the instance is missing required license. -->
<!-- Normally this should NOT be a blocker to use the feature, it's just a warning message. -->
<!-- We can force to show this warning by passing props:show as true -->
<template>
  <div
    v-if="show || instanceMissingLicense"
    :class="['text-accent cursor-pointer', customClass]"
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
        <slot name="default">
          <heroicons-outline:exclamation class="text-warning w-5 h-5" />
        </slot>
      </template>
      <span class="w-56 text-sm">
        {{
          tooltip ||
          $t("subscription.instance-assignment.missing-license-for-feature", {
            feature: $t(
              `dynamic.subscription.features.${feature.split(".").join("-")}.title`
            ).toLowerCase(),
          })
        }}
      </span>
    </NTooltip>
  </div>
  <InstanceAssignment
    v-if="(instanceMissingLicense || show) && canManageSubscription"
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { reactive, computed } from "vue";
import { useSubscriptionV1Store } from "@/store";
import type { FeatureType } from "@/types";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto/api/v1alpha/instance_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import InstanceAssignment from "../InstanceAssignment.vue";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const props = withDefaults(
  defineProps<{
    show?: boolean;
    feature: FeatureType;
    instance?: Instance | InstanceResource;
    customClass?: string;
    tooltip?: string;
  }>(),
  {
    show: false,
    instance: undefined,
    customClass: "",
    tooltip: "",
  }
);

const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
});

const subscriptionStore = useSubscriptionV1Store();

const instanceMissingLicense = computed(() => {
  return subscriptionStore.instanceMissingLicense(
    props.feature,
    props.instance
  );
});

const canManageSubscription = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.instances.update");
});
</script>
