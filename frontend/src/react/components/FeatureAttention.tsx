import { CircleAlert, Info } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { useActuatorV1Store, useSubscriptionV1Store } from "@/store";
import { ENTERPRISE_INQUIRE_LINK, instanceLimitFeature } from "@/types";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { autoSubscriptionRoute, hasWorkspacePermissionV2 } from "@/utils";
import { useVueState } from "../hooks/useVueState";

export function FeatureAttention({
  feature,
  description: descriptionProp,
}: {
  feature: PlanFeature;
  description?: string;
}) {
  const { t } = useTranslation();
  const subscriptionStore = useSubscriptionV1Store();
  const actuatorStore = useActuatorV1Store();

  const hasFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(feature)
  );
  const existInstanceWithoutLicense = useVueState(
    () =>
      actuatorStore.totalInstanceCount > actuatorStore.activatedInstanceCount &&
      instanceLimitFeature.has(feature)
  );

  const show = !hasFeature || existInstanceWithoutLicense;
  if (!show) return null;

  const isWarning = !hasFeature;
  const featureKey = PlanFeature[feature].split(".").join("-");

  const title = t(`dynamic.subscription.features.${featureKey}.title`);

  const featureDesc =
    descriptionProp || t(`dynamic.subscription.features.${featureKey}.desc`);

  let descriptionText: string;
  if (!hasFeature) {
    const startTrial = subscriptionStore.isTrialing
      ? ""
      : t("subscription.trial-for-days", {
          days: subscriptionStore.trialingDays,
        });
    const requiredPlan = subscriptionStore.getMinimumRequiredPlan(feature);
    if (
      requiredPlan === PlanType.FREE &&
      subscriptionStore.hasFeature(feature)
    ) {
      descriptionText = `${featureDesc}\n${startTrial}`;
    } else {
      const trialText = t("subscription.required-plan-with-trial", {
        requiredPlan: t(
          `subscription.plan.${PlanType[requiredPlan].toLowerCase()}.title`
        ),
        startTrial,
      });
      descriptionText = `${featureDesc}\n${trialText}`;
    }
  } else {
    const attention = t(
      "subscription.instance-assignment.missing-license-attention"
    );
    descriptionText = `${featureDesc}\n${attention}`;
  }

  const hasPermission = hasWorkspacePermissionV2("bb.settings.set");

  let actionText = "";
  if (hasPermission) {
    if (!hasFeature) {
      actionText = t("subscription.request-n-days-trial", {
        days: subscriptionStore.trialingDays,
      });
    } else if (hasWorkspacePermissionV2("bb.instances.update")) {
      actionText = t("subscription.instance-assignment.assign-license");
    }
  }

  const onAction = () => {
    if (!hasFeature) {
      // Vue version shows WeChat QR modal for zh-CN, but that's a Vue
      // component. Open the inquiry link for all locales as a fallback.
      window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
      return;
    }
    // Vue version opens InstanceAssignment drawer here, which is a Vue
    // component not yet migrated. Navigate to the subscription page instead,
    // matching the Vue fallback behavior.
    router.push(autoSubscriptionRoute());
  };

  const Icon = isWarning ? CircleAlert : Info;

  return (
    <div
      className={`flex items-start gap-3 rounded-md border px-4 py-3 ${
        isWarning
          ? "border-yellow-300 bg-yellow-50"
          : "border-blue-200 bg-blue-50"
      }`}
    >
      <Icon
        className={`w-5 h-5 mt-0.5 shrink-0 ${
          isWarning ? "text-yellow-600" : "text-blue-600"
        }`}
      />
      <div className="flex-1 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <p
            className={`font-medium text-sm ${
              isWarning ? "text-yellow-800" : "text-blue-800"
            }`}
          >
            {title}
          </p>
          <p
            className={`text-sm whitespace-pre-line mt-1 ${
              isWarning ? "text-yellow-700" : "text-blue-700"
            }`}
          >
            {descriptionText}
          </p>
        </div>
        {actionText && (
          <Button
            variant="outline"
            size="sm"
            className="shrink-0 whitespace-nowrap"
            onClick={onAction}
          >
            {actionText}
          </Button>
        )}
      </div>
    </div>
  );
}
