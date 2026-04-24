import { CircleAlert, Info } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { useActuatorV1Store, useSubscriptionV1Store } from "@/store";
import { ENTERPRISE_INQUIRE_LINK, instanceLimitFeature } from "@/types";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { autoSubscriptionRoute, hasWorkspacePermissionV2 } from "@/utils";
import { useVueState } from "../hooks/useVueState";
import { InstanceAssignmentSheet } from "./InstanceAssignmentSheet";

export function FeatureAttention({
  feature,
  description: descriptionProp,
  instance,
}: {
  feature: PlanFeature;
  description?: string;
  instance?: Instance | InstanceResource;
}) {
  const { t } = useTranslation();
  const subscriptionStore = useSubscriptionV1Store();
  const actuatorStore = useActuatorV1Store();
  const [showInstanceAssignment, setShowInstanceAssignment] = useState(false);

  const hasFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(feature)
  );
  const instanceMissingLicense = useVueState(() =>
    subscriptionStore.instanceMissingLicense(feature, instance)
  );
  const existInstanceWithoutLicense = useVueState(
    () =>
      actuatorStore.totalInstanceCount > actuatorStore.activatedInstanceCount &&
      instanceLimitFeature.has(feature)
  );

  const show =
    !hasFeature ||
    instanceMissingLicense ||
    (!instance && existInstanceWithoutLicense);
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
      window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
      return;
    }
    if (instanceMissingLicense || existInstanceWithoutLicense) {
      setShowInstanceAssignment(true);
      return;
    }
    router.push(autoSubscriptionRoute());
  };

  const Icon = isWarning ? CircleAlert : Info;

  return (
    <>
      <Alert variant={isWarning ? "warning" : "info"} showIcon={false}>
        <Icon className="size-5 mt-0.5 shrink-0" />
        <div className="flex-1 flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <AlertTitle>{title}</AlertTitle>
            <AlertDescription className="mt-0 whitespace-pre-line">
              {descriptionText}
            </AlertDescription>
          </div>
          {actionText && (
            <div className="flex justify-end">
              <Button
                variant="outline"
                size="sm"
                className="shrink-0 whitespace-nowrap"
                onClick={onAction}
              >
                {actionText}
              </Button>
            </div>
          )}
        </div>
      </Alert>
      <InstanceAssignmentSheet
        open={showInstanceAssignment}
        selectedInstanceList={instance ? [instance.name] : []}
        onOpenChange={setShowInstanceAssignment}
      />
    </>
  );
}
