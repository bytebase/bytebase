<template>
  <BBModal v-if="open" :title="title" @close="$emit('cancel')">
    <div class="min-w-0 md:min-w-[400px] max-w-4xl">
      <div class="flex items-start space-x-2 mt-3">
        <div class="flex items-center">
          <heroicons-solid:lock-closed
            v-if="instanceMissingLicense"
            class="text-accent w-6 h-6"
          />
          <heroicons-solid:sparkles v-else class="h-6 w-6 text-accent" />
        </div>
        <h3
          id="modal-headline"
          class="flex self-center text-lg leading-6 font-medium text-gray-900"
        >
          {{ $t(`subscription.features.${featureKey}.title`) }}
        </h3>
      </div>
      <div class="mt-5">
        <p class="whitespace-pre-wrap">
          {{ $t(`subscription.features.${featureKey}.desc`) }}
        </p>
      </div>
      <div class="mt-3">
        <p class="whitespace-pre-wrap">
          <i18n-t
            v-if="instanceMissingLicense"
            keypath="subscription.instance-assignment.missing-license-attention"
          />
          <template v-else-if="subscriptionStore.canTrial">
            <i18n-t
              v-if="isRequiredInPlan"
              keypath="subscription.required-plan-with-trial"
            >
              <template #requiredPlan>
                <span class="font-bold text-accent">
                  {{
                    $t(
                      `subscription.plan.${planTypeToString(
                        requiredPlan
                      )}.title`
                    )
                  }}
                </span>
              </template>
              <template v-if="!hasPermission" #startTrial>
                {{ $t("subscription.contact-to-upgrade") }}
              </template>
              <template
                v-else-if="subscriptionStore.canUpgradeTrial"
                #startTrial
              >
                {{ $t("subscription.upgrade-trial") }}
              </template>
              <template v-else #startTrial>
                {{
                  $t("subscription.trial-for-days", {
                    days: subscriptionStore.trialingDays,
                  })
                }}
              </template>
            </i18n-t>
            <i18n-t v-else keypath="subscription.trial-for-days">
              <template #days>
                {{ subscriptionStore.trialingDays }}
              </template>
            </i18n-t>
          </template>
          <i18n-t v-else keypath="subscription.require-subscription">
            <template #requiredPlan>
              <span class="font-bold text-accent">
                {{
                  $t(
                    `subscription.plan.${planTypeToString(requiredPlan)}.title`
                  )
                }}
              </span>
            </template>
          </i18n-t>
        </p>
      </div>
      <div class="mt-7 flex justify-end space-x-2">
        <button
          v-if="!hasPermission"
          type="button"
          class="btn-primary"
          @click.prevent="$emit('cancel')"
        >
          {{ $t("common.ok") }}
        </button>
        <template v-else-if="subscriptionStore.canTrial">
          <button
            v-if="subscriptionStore.canUpgradeTrial"
            type="button"
            class="btn-primary"
            @click.prevent="trialSubscription"
          >
            {{ $t("subscription.upgrade-trial-button") }}
          </button>
          <button
            v-else
            type="button"
            class="btn-primary"
            @click.prevent="trialSubscription"
          >
            {{
              $t("subscription.start-n-days-trial", {
                days: subscriptionStore.trialingDays,
              })
            }}
          </button>
        </template>
        <button v-else type="button" class="btn-primary" @click.prevent="ok">
          {{
            instanceMissingLicense
              ? $t("subscription.instance-assignment.assign-license")
              : $t("common.learn-more")
          }}
        </button>
      </div>
    </div>
  </BBModal>
  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { PropType, computed, reactive } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
  useSubscriptionV1Store,
  useCurrentUserV1,
  pushNotification,
} from "@/store";
import { FeatureType, planTypeToString } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { Instance } from "@/types/proto/v1/instance_service";
import { hasWorkspacePermissionV1 } from "@/utils";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const props = defineProps({
  open: {
    required: true,
    type: Boolean,
  },
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

const emit = defineEmits(["cancel"]);
const { t } = useI18n();
const router = useRouter();

const subscriptionStore = useSubscriptionV1Store();
const hasPermission = hasWorkspacePermissionV1(
  "bb.permission.workspace.manage-subscription",
  useCurrentUserV1().value.userRole
);

const instanceMissingLicense = computed(() => {
  return subscriptionStore.instanceMissingLicense(
    props.feature,
    props.instance
  );
});

const title = computed(() => {
  if (instanceMissingLicense.value) {
    return t("subscription.instance-assignment.require-license");
  }
  return t("subscription.disabled-feature");
});

const ok = () => {
  if (instanceMissingLicense.value) {
    state.showInstanceAssignmentDrawer = true;
  } else {
    router.push({
      name: "setting.workspace.subscription",
    });
  }
  emit("cancel");
};

const isRequiredInPlan = Array.isArray(
  subscriptionStore.featureMatrix.get(props.feature)
);
const requiredPlan = subscriptionStore.getMinimumRequiredPlan(props.feature);

const featureKey = props.feature.split(".").join("-");

const trialSubscription = () => {
  const isUpgrade = subscriptionStore.canUpgradeTrial;
  subscriptionStore.trialSubscription(PlanType.ENTERPRISE).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.success"),
      description: isUpgrade
        ? t("subscription.successfully-upgrade-trial", {
            plan: t(
              `subscription.plan.${planTypeToString(
                subscriptionStore.currentPlan
              )}.title`
            ),
          })
        : t("subscription.successfully-start-trial", {
            days: subscriptionStore.trialingDays,
          }),
    });
    emit("cancel");
  });
};
</script>

<style scoped>
@media (min-width: 768px) {
  .md\:min-w-400 {
    min-width: 400px;
  }
}
</style>
