<template>
  <div v-if="subscriptionStore.currentPlan === PlanType.FREE" class="my-3">
    <dt class="text-main">
      {{ $t("subscription.max-instance-count") }}
    </dt>
    <dd class="mt-1 text-4xl flex items-center gap-x-2 cursor-pointer group">
      <span class="group-hover:underline">
        {{ subscriptionStore.instanceCountLimit }}
      </span>
    </dd>
  </div>
  <div v-else class="my-3">
    <dt class="text-main">
      {{ $t("subscription.instance-assignment.used-and-total-license") }}
    </dt>
    <dd
      class="mt-1 text-4xl flex items-center gap-x-2 cursor-pointer group"
      @click="state.showInstanceAssignmentDrawer = true"
    >
      <span class="group-hover:underline">{{ activateLicenseCount }}</span>
      <span class="text-xl">/</span>
      <span class="group-hover:underline">{{ totalLicenseCount }}</span>
      <heroicons-outline:pencil class="h-6 w-6" />
    </dd>
  </div>

  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import InstanceAssignment from "@/components/InstanceAssignment.vue";
import {
  useInstanceV1List,
  useInstanceV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
});
const subscriptionStore = useSubscriptionV1Store();
const instanceV1Store = useInstanceV1Store();
// Prepare instance list.
useInstanceV1List();

const { instanceLicenseCount } = storeToRefs(subscriptionStore);

const totalLicenseCount = computed((): string => {
  if (instanceLicenseCount.value === Number.MAX_VALUE) {
    return t("subscription.unlimited");
  }
  return `${instanceLicenseCount.value}`;
});

const activateLicenseCount = computed((): string => {
  return `${instanceV1Store.activateInstanceCount}`;
});
</script>
