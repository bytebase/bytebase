<template>
  <div
    v-if="instanceMissingLicense"
    class="tooltip-wrapper cursor-pointer"
    @click="state.showInstanceAssignmentDrawer = true"
  >
    <heroicons-solid:lock-closed class="text-accent w-5 h-5" />
    <span class="w-56 text-sm -translate-y-full -translate-x-1/3 tooltip">
      {{ $t("subscription.instance-assignment.missing-license-attention") }}
    </span>
  </div>
  <router-link
    v-else-if="!hasFeature"
    to="/setting/subscription"
    exact-active-class=""
  >
    <heroicons-solid:sparkles class="text-accent w-5 h-5" />
  </router-link>
  <InstanceAssignment
    v-if="!hasFeature"
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { reactive, PropType, computed } from "vue";
import { FeatureType } from "@/types";
import { useSubscriptionV1Store } from "@/store";
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
