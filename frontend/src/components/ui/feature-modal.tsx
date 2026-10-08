import { Code, ConnectError } from "@connectrpc/connect";
import { LoaderCircle, Lock, Sparkles, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { captureFeatureGateMetric } from "@/app/analytics/feature-gate";
import { router } from "@/app/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/app/router/handles";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog";
import { useSubscriptionState } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { autoSubscriptionRoute, hasWorkspacePermissionV2 } from "@/utils";

type Props = {
  readonly open: boolean;
  readonly feature: PlanFeature | undefined;
  readonly instance?: Instance | InstanceResource;
  readonly onOpenChange: (open: boolean) => void;
  readonly onFeatureUnlocked?: () => void;
};

const planLabel: Record<number, string> = {
  [PlanType.FREE]: "free",
  [PlanType.TEAM]: "team",
  [PlanType.ENTERPRISE]: "enterprise",
};

/**
 * Subscription paywall dialog shown when the current plan doesn't include
 * a requested feature. Three copy variants:
 *  - Instance missing license → lock icon, "assign license" CTA.
 *  - Required plan above FREE → plan name + trial or contact-admin line.
 *  - Required plan equals FREE → trial-for-days line.
 */
export function FeatureModal({
  open,
  feature,
  instance,
  onOpenChange,
  onFeatureUnlocked,
}: Props) {
  const { t } = useTranslation();
  const { canStartTrial, showTrial, startTrial, trialingDays } =
    useSubscriptionState();
  const canManageSettings = hasWorkspacePermissionV2("bb.settings.set");
  const canManageSubscription = hasWorkspacePermissionV2(
    "bb.subscription.manage"
  );
  const wasOpenRef = useRef(false);
  const [startingTrial, setStartingTrial] = useState(false);
  const [trialRejected, setTrialRejected] = useState(false);

  const resolvedFeature = feature ?? PlanFeature.FEATURE_UNSPECIFIED;
  const instanceMissingLicense = useAppStore((state) =>
    state.instanceMissingLicense(resolvedFeature, instance)
  );
  const requiredPlan = useAppStore((state) =>
    state.getMinimumRequiredPlan(resolvedFeature)
  );

  useEffect(() => {
    const isOpen = open && !!feature;
    if (isOpen && !wasOpenRef.current) {
      setTrialRejected(false);
      captureFeatureGateMetric(
        "locked feature clicked",
        resolvedFeature,
        instance
      );
    }
    wasOpenRef.current = isOpen;
  }, [feature, instance, open, resolvedFeature]);

  if (!feature) {
    return null;
  }

  const featureKey = PlanFeature[feature].split(".").join("-");
  const title = instanceMissingLicense
    ? t("subscription.instance-assignment.require-license")
    : t("subscription.disabled-feature");

  const close = () => onOpenChange(false);

  const canOfferTrial =
    canStartTrial && !instanceMissingLicense && !trialRejected;
  const hasPermission = canOfferTrial
    ? canManageSubscription
    : canManageSettings || (trialRejected && canManageSubscription);

  const confirmLabel = instanceMissingLicense
    ? t("subscription.instance-assignment.assign-license")
    : t("common.learn-more");

  const handleConfirm = () => {
    if (instanceMissingLicense) {
      void router.push({
        name: INSTANCE_ROUTE_DASHBOARD,
        query: {
          assignLicense: "1",
          instances: instance?.name,
        },
      });
    } else {
      void router.push(autoSubscriptionRoute());
    }
    close();
  };

  const handleStartTrial = async () => {
    setStartingTrial(true);
    try {
      await startTrial();
      close();
      onFeatureUnlocked?.();
    } catch (error) {
      if (ConnectError.from(error).code === Code.FailedPrecondition) {
        setTrialRejected(true);
      }
    } finally {
      setStartingTrial(false);
    }
  };

  const requiredPlanLabel = t(
    `subscription.plan.${planLabel[requiredPlan] ?? "enterprise"}.title`
  );
  const startTrialLabel = hasPermission
    ? t("subscription.trial-for-days", { days: trialingDays })
    : t("subscription.contact-to-upgrade");

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen, details) => {
        if (startingTrial) {
          details.cancel();
          return;
        }
        onOpenChange(nextOpen);
      }}
    >
      <DialogContent className="max-w-2xl">
        <div>
          <div className="flex items-center justify-between border-b pb-2 mb-4">
            <DialogTitle className="text-base font-medium">{title}</DialogTitle>
            <DialogClose
              aria-label={t("common.close")}
              disabled={startingTrial}
              className="rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
            >
              <X className="size-4" />
            </DialogClose>
          </div>
          <div className="flex items-start gap-x-2">
            <div className="flex items-center">
              {instanceMissingLicense ? (
                <Lock className="size-6 text-accent" />
              ) : (
                <Sparkles className="size-6 text-accent" />
              )}
            </div>
            <h3 className="flex self-center text-lg leading-6 font-medium">
              {t(`dynamic.subscription.features.${featureKey}.title`)}
            </h3>
          </div>

          <div className="mt-4">
            <p className="whitespace-pre-wrap">
              {t(`dynamic.subscription.features.${featureKey}.desc`)}
            </p>
          </div>

          <div className="mt-3">
            <p className="whitespace-pre-wrap">
              {instanceMissingLicense
                ? t(
                    "subscription.instance-assignment.missing-license-attention"
                  )
                : requiredPlan !== PlanType.FREE
                  ? t("subscription.required-plan-with-trial", {
                      requiredPlan: requiredPlanLabel,
                      startTrial: startTrialLabel,
                    })
                  : t("subscription.trial-for-days", {
                      days: trialingDays,
                    })}
            </p>
          </div>

          <div className="mt-7 flex justify-end gap-x-2">
            {!hasPermission ? (
              <Button variant="default" onClick={close}>
                {t("common.ok")}
              </Button>
            ) : canOfferTrial ? (
              <Button
                variant="default"
                disabled={startingTrial}
                onClick={() => void handleStartTrial()}
              >
                {startingTrial && <LoaderCircle className="animate-spin" />}
                {t("subscription.plan.try")}
              </Button>
            ) : showTrial && !instanceMissingLicense ? (
              <Button
                variant="default"
                onClick={() => {
                  window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
                  // Close so the paywall doesn't linger after the inquiry
                  // tab opens.
                  close();
                }}
              >
                {t("subscription.request-n-days-trial", {
                  days: trialingDays,
                })}
              </Button>
            ) : (
              <Button variant="default" onClick={handleConfirm}>
                {confirmLabel}
              </Button>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
