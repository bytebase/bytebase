<template>
  <div v-bind="$attrs">
    <dt class="text-main">
      {{
        subscriptionStore.currentPlan === PlanType.FREE
          ? $t("subscription.max-instance-count")
          : $t("subscription.instance-assignment.used-and-total-license")
      }}
    </dt>
    <div class="mt-1 text-4xl flex items-center gap-2">
      <span v-if="subscriptionStore.currentPlan === PlanType.FREE">
        {{ subscriptionStore.instanceCountLimit }}
      </span>
      <template v-else>
        <span
          >{{ activateLicenseCount }}
          <span class="font-mono text-gray-500">/</span>
          {{ totalLicenseCount }}</span
        >
        <NButton text size="large">
          <template #icon>
            <PencilIcon
              class="h-8 w-8"
              @click="state.showInstanceAssignmentDrawer = true"
            />
          </template>
        </NButton>
      </template>
    </div>
  </div>

  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import InstanceAssignment from "@/components/InstanceAssignment.vue";
import { useActuatorV1Store, useSubscriptionV1Store } from "@/store";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
});
const subscriptionStore = useSubscriptionV1Store();
const actuatorStore = useActuatorV1Store();

const { instanceLicenseCount } = storeToRefs(subscriptionStore);

const totalLicenseCount = computed((): string => {
  if (instanceLicenseCount.value === Number.MAX_VALUE) {
    return t("common.unlimited");
  }
  return `${instanceLicenseCount.value}`;
});

const activateLicenseCount = computed((): string => {
  return `${actuatorStore.activatedInstanceCount}`;
});
</script>
