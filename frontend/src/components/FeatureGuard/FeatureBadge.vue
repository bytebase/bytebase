<template>
  <div
    v-if="instanceMissingLicense"
    :class="['text-accent cursor-pointer', customClass]"
    @click="state.showInstanceAssignmentDrawer = true"
  >
    <NTooltip :show-arrow="true">
      <template #trigger>
        <heroicons-solid:lock-closed class="text-accent w-5 h-5" />
      </template>
      <span class="w-56 text-sm">
        {{ $t("subscription.instance-assignment.missing-license-attention") }}
      </span>
    </NTooltip>
  </div>
  <template v-else-if="!hasFeature">
    <NTooltip :show-arrow="true">
      <template #trigger>
        <router-link
          v-if="clickable"
          to="/setting/subscription"
          exact-active-class=""
        >
          <heroicons-solid:sparkles class="text-accent w-5 h-5" />
        </router-link>
        <span v-else>
          <heroicons-solid:sparkles class="text-accent w-5 h-5" />
        </span>
      </template>
      <span class="w-56 text-sm">
        {{
          $t("subscription.require-subscription", {
            requiredPlan: $t(
              `subscription.plan.${planTypeToString(
                subscriptionStore.getMinimumRequiredPlan(feature)
              )}.title`
            ),
          })
        }}
      </span>
    </NTooltip>
  </template>
  <InstanceAssignment
    v-if="instanceMissingLicense"
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { reactive, PropType, computed } from "vue";
import { useSubscriptionV1Store } from "@/store";
import { FeatureType, planTypeToString } from "@/types";
import { Instance } from "@/types/proto/v1/instance_service";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const props = defineProps({
  feature: {
    required: true,
    type: String as PropType<FeatureType>,
  },
  instance: {
    type: Object as PropType<Instance>,
    default: undefined,
  },
  customClass: {
    require: false,
    default: "",
    type: String,
  },
  clickable: {
    type: Boolean,
    default: false,
  },
});

const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
});

const subscriptionStore = useSubscriptionV1Store();

const hasFeature = computed(() => {
  return subscriptionStore.hasInstanceFeature(props.feature, props.instance);
});

const instanceMissingLicense = computed(() => {
  return subscriptionStore.instanceMissingLicense(
    props.feature,
    props.instance
  );
});
</script>
