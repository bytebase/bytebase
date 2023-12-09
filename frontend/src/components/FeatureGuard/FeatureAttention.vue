<template>
  <BBAttention
    v-if="!hasFeature"
    :class="customClass"
    :style="`WARN`"
    :title="$t(`subscription.features.${featureKey}.title`)"
    :description="descriptionText"
    :action-text="actionText"
    @click-action="onClick"
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
import { reactive, PropType, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useLanguage } from "@/composables/useLanguage";
import { useSubscriptionV1Store, useCurrentUserV1 } from "@/store";
import {
  FeatureType,
  planTypeToString,
  ENTERPRISE_INQUIRE_LINK,
} from "@/types";
import { Instance } from "@/types/proto/v1/instance_service";
import { hasWorkspacePermissionV1 } from "@/utils";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
  showQRCodeModal: boolean;
}

const props = defineProps({
  feature: {
    required: true,
    type: String as PropType<FeatureType>,
  },
  description: {
    require: false,
    default: "",
    type: String,
  },
  customClass: {
    require: false,
    default: "",
    type: String,
  },
  instance: {
    type: Object as PropType<Instance>,
    default: undefined,
  },
});

const router = useRouter();
const { t } = useI18n();
const { locale } = useLanguage();

const subscriptionStore = useSubscriptionV1Store();
const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
  showQRCodeModal: false,
});

const hasPermission = hasWorkspacePermissionV1(
  "bb.permission.workspace.manage-subscription",
  useCurrentUserV1().value.userRole
);

const hasFeature = computed(() => {
  return subscriptionStore.hasInstanceFeature(props.feature, props.instance);
});

const instanceMissingLicense = computed(() => {
  return subscriptionStore.instanceMissingLicense(
    props.feature,
    props.instance
  );
});

const actionText = computed(() => {
  if (!hasPermission) {
    return "";
  }
  if (instanceMissingLicense.value) {
    return t("subscription.instance-assignment.assign-license");
  }
  if (!subscriptionStore.canTrial) {
    return t("subscription.upgrade");
  }
  if (subscriptionStore.canUpgradeTrial) {
    return t("subscription.upgrade-trial-button");
  }
  return t("subscription.request-n-days-trial", {
    days: subscriptionStore.trialingDays,
  });
});

const featureKey = props.feature.split(".").join("-");

const descriptionText = computed(() => {
  let description = props.description;
  if (!description) {
    description = t(`subscription.features.${featureKey}.desc`);
  }

  if (instanceMissingLicense.value) {
    const attention = t(
      "subscription.instance-assignment.missing-license-attention"
    );
    return `${description}\n${attention}`;
  }

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

  const requiredPlan = subscriptionStore.getMinimumRequiredPlan(props.feature);
  const trialText = t("subscription.required-plan-with-trial", {
    requiredPlan: t(
      `subscription.plan.${planTypeToString(requiredPlan)}.title`
    ),
    startTrial: startTrial,
  });

  return `${description}\n${trialText}`;
});

const onClick = () => {
  if (instanceMissingLicense.value) {
    state.showInstanceAssignmentDrawer = true;
  } else if (subscriptionStore.canTrial) {
    if (locale.value === "zh-CN") {
      state.showQRCodeModal = true;
    } else {
      window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
    }
  } else {
    router.push({ name: "setting.workspace.subscription" });
  }
};
</script>
