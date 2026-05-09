import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import {
  useServerState,
  useSubscriptionState,
} from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { router } from "@/router";
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
  const { trialingDays, isTrialing } = useSubscriptionState();
  const { totalInstanceCount, activatedInstanceCount } = useServerState();
  const [showInstanceAssignment, setShowInstanceAssignment] = useState(false);

  const hasFeature = useAppStore((state) => state.hasInstanceFeature(feature));
  const instanceMissingLicense = useAppStore((state) =>
    state.instanceMissingLicense(feature, instance)
  );
  const hasUnifiedInstanceLicense = useAppStore((state) =>
    state.hasUnifiedInstanceLicense()
  );
  const requiredPlan = useAppStore((state) =>
    state.getMinimumRequiredPlan(feature)
  );
  const featureIncludedInPlan = useAppStore((state) =>
    state.hasFeature(feature)
  );
  const existInstanceWithoutLicense =
    !hasUnifiedInstanceLicense &&
    totalInstanceCount > activatedInstanceCount &&
    instanceLimitFeature.has(feature);

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
    const startTrial = isTrialing
      ? ""
      : t("subscription.trial-for-days", {
          days: trialingDays,
        });
    if (requiredPlan === PlanType.FREE && featureIncludedInPlan) {
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
        days: trialingDays,
      });
    } else if (
      !hasUnifiedInstanceLicense &&
      hasWorkspacePermissionV2("bb.instances.update")
    ) {
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

  return (
    <>
      <Alert
        variant={isWarning ? "warning" : "info"}
        title={title}
        description={
          <span className="whitespace-pre-line">{descriptionText}</span>
        }
      >
        {actionText && (
          <div className="mt-3 flex justify-end">
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
      </Alert>
      {!hasUnifiedInstanceLicense && (
        <InstanceAssignmentSheet
          open={showInstanceAssignment}
          selectedInstanceList={instance ? [instance.name] : []}
          onOpenChange={setShowInstanceAssignment}
        />
      )}
    </>
  );
}
