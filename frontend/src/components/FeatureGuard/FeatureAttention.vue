<template>
  <BBAttention
    v-if="show"
    v-bind="$attrs"
    :type="hasFeature ? type : 'warning'"
    :title="$t(`dynamic.subscription.features.${featureKey}.title`)"
    :description="descriptionText"
    :action-text="actionText"
    @click="onClick"
  />

  <WeChatQRModal
    v-if="state.showQRCodeModal"
    :title="$t('subscription.request-with-qr')"
    @close="state.showQRCodeModal = false"
  />
  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import { useLanguage } from "@/composables/useLanguage";
import { useSubscriptionV1Store, useActuatorV1Store } from "@/store";
import type { FeatureType } from "@/types";
import { planTypeToString, ENTERPRISE_INQUIRE_LINK } from "@/types";
import { instanceLimitFeature } from "@/types";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto/v1/instance_service";
import { autoSubscriptionRoute, hasWorkspacePermissionV2 } from "@/utils";
import InstanceAssignment from "../InstanceAssignment.vue";
import WeChatQRModal from "../WeChatQRModal.vue";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
  showQRCodeModal: boolean;
}

const props = withDefaults(
  defineProps<{
    feature: FeatureType;
    description?: string;
    type?: "info" | "warning" | "error";
    instance?: Instance | InstanceResource;
  }>(),
  {
    type: "info",
    description: "",
    instance: undefined,
  }
);

const router = useRouter();
const { t } = useI18n();
const { locale } = useLanguage();
const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();

const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
  showQRCodeModal: false,
});

const hasPermission = computed(() =>
  hasWorkspacePermissionV2("bb.settings.set")
);

const canManageInstanceLicense = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.instances.update");
});

const hasFeature = computed(() => {
  return subscriptionStore.hasInstanceFeature(props.feature);
});

const instanceMissingLicense = computed(() => {
  return subscriptionStore.instanceMissingLicense(
    props.feature,
    props.instance
  );
});

const existInstanceWithoutLicense = computed(() => {
  return (
    actuatorStore.totalInstanceCount > actuatorStore.activatedInstanceCount &&
    instanceLimitFeature.has(props.feature)
  );
});

const show = computed(() => {
  if (!hasFeature.value) {
    // show missing feature attention.
    return true;
  }
  if (instanceMissingLicense.value || existInstanceWithoutLicense.value) {
    // show missing instance license attention.
    return true;
  }
  return false;
});

const actionText = computed(() => {
  if (!hasPermission.value) {
    return "";
  }

  if (!hasFeature.value) {
    return t("subscription.request-n-days-trial", {
      days: subscriptionStore.trialingDays,
    });
  }

  if (!canManageInstanceLicense.value) {
    return "";
  }
  return t("subscription.instance-assignment.assign-license");
});

const featureKey = props.feature.split(".").join("-");

const descriptionText = computed(() => {
  let description = props.description;
  if (!description) {
    description = t(`dynamic.subscription.features.${featureKey}.desc`);
  }

  if (!hasFeature.value) {
    const startTrial = subscriptionStore.canUpgradeTrial
      ? t("subscription.upgrade-trial")
      : subscriptionStore.isTrialing
        ? ""
        : t("subscription.trial-for-days", {
            days: subscriptionStore.trialingDays,
          });
    if (!Array.isArray(subscriptionStore.featureMatrix.get(props.feature))) {
      return `${description}\n${startTrial}`;
    }

    const requiredPlan = subscriptionStore.getMinimumRequiredPlan(
      props.feature
    );
    const trialText = t("subscription.required-plan-with-trial", {
      requiredPlan: t(
        `subscription.plan.${planTypeToString(requiredPlan)}.title`
      ),
      startTrial: startTrial,
    });

    return `${description}\n${trialText}`;
  }

  const attention = t(
    "subscription.instance-assignment.missing-license-attention"
  );
  return `${description}\n${attention}`;
});

const onClick = () => {
  if (!hasFeature.value) {
    if (locale.value === "zh-CN") {
      state.showQRCodeModal = true;
    } else {
      window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
    }
    return;
  }
  if (instanceMissingLicense.value || existInstanceWithoutLicense.value) {
    state.showInstanceAssignmentDrawer = true;
    return;
  }
  router.push(autoSubscriptionRoute(router));
};
</script>
