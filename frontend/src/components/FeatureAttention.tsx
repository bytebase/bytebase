import { Code, ConnectError } from "@connectrpc/connect";
import { LoaderCircle } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useServerState, useSubscriptionState } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
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

// `compact` is one row for a section that already names and describes the
// feature: what the plan requires, and a short action.
export function FeatureAttention({
  feature,
  description: descriptionProp,
  instance,
  compact = false,
}: {
  feature: PlanFeature;
  description?: string;
  instance?: Instance | InstanceResource;
  compact?: boolean;
}) {
  const { t } = useTranslation();
  const { canStartTrial, isTrialing, startTrial, trialingDays } =
    useSubscriptionState();
  const { totalInstanceCount, activatedInstanceCount } = useServerState();
  const [showInstanceAssignment, setShowInstanceAssignment] = useState(false);
  const [startingTrial, setStartingTrial] = useState(false);
  const [trialRejected, setTrialRejected] = useState(false);

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

  // What the plan or the instance license lacks, without the feature's own
  // description.
  let requirementText: string;
  if (!hasFeature) {
    const startTrial = isTrialing
      ? ""
      : t("subscription.trial-for-days", {
          days: trialingDays,
        });
    requirementText =
      requiredPlan === PlanType.FREE && featureIncludedInPlan
        ? startTrial
        : t("subscription.required-plan-with-trial", {
            requiredPlan: t(
              `subscription.plan.${PlanType[requiredPlan].toLowerCase()}.title`
            ),
            startTrial,
          });
  } else {
    requirementText = t(
      "subscription.instance-assignment.missing-license-attention"
    );
  }
  const descriptionText = `${featureDesc}\n${requirementText}`;

  const canManageSettings = hasWorkspacePermissionV2("bb.settings.set");
  const canManageSubscription = hasWorkspacePermissionV2(
    "bb.subscription.manage"
  );
  const canOfferTrial = canStartTrial && !trialRejected;
  const hasPermission = canOfferTrial
    ? canManageSubscription
    : canManageSettings || (trialRejected && canManageSubscription);

  let actionText = "";
  if (hasPermission) {
    if (!hasFeature) {
      actionText = canOfferTrial
        ? t("subscription.plan.try")
        : trialRejected
          ? t("common.learn-more")
          : compact
            ? t("subscription.try-for-free")
            : t("subscription.request-n-days-trial", {
                days: trialingDays,
              });
    } else if (
      !hasUnifiedInstanceLicense &&
      hasWorkspacePermissionV2("bb.instances.update")
    ) {
      actionText = t("subscription.instance-assignment.assign-license");
    }
  }

  const onAction = async () => {
    if (!hasFeature) {
      if (canOfferTrial) {
        setStartingTrial(true);
        try {
          await startTrial();
        } catch (error) {
          if (ConnectError.from(error).code === Code.FailedPrecondition) {
            setTrialRejected(true);
          }
        } finally {
          setStartingTrial(false);
        }
        return;
      }
      if (trialRejected) {
        void router.push(autoSubscriptionRoute());
        return;
      }
      window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
      return;
    }
    if (instanceMissingLicense || existInstanceWithoutLicense) {
      setShowInstanceAssignment(true);
      return;
    }
    router.push(autoSubscriptionRoute());
  };

  const actionButton = actionText && (
    <Button
      appearance={compact ? "outline" : "solid"}
      size="sm"
      className="shrink-0 whitespace-nowrap"
      disabled={startingTrial}
      onClick={() => void onAction()}
    >
      {startingTrial && <LoaderCircle className="animate-spin" />}
      {actionText}
    </Button>
  );

  return (
    <>
      {compact ? (
        // An upsell is not a warning: the info tone, with the body in the
        // normal text color so the sentence reads first.
        <Alert variant="info" className="items-center rounded-sm shadow-none">
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            <p className="min-w-0 flex-1 text-control">{requirementText}</p>
            {actionButton}
          </div>
        </Alert>
      ) : (
        <Alert
          variant={isWarning ? "warning" : "info"}
          title={title}
          description={
            <span className="whitespace-pre-line">{descriptionText}</span>
          }
        >
          {actionButton && (
            <div className="mt-3 flex justify-end">{actionButton}</div>
          )}
        </Alert>
      )}
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
