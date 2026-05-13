import { Lock, Sparkles, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { useSubscriptionState } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { router } from "@/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
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
export function FeatureModal({ open, feature, instance, onOpenChange }: Props) {
  const { t } = useTranslation();
  const { showTrial, trialingDays } = useSubscriptionState();
  const hasPermission = hasWorkspacePermissionV2("bb.settings.set");

  const resolvedFeature = feature ?? PlanFeature.FEATURE_UNSPECIFIED;
  const instanceMissingLicense = useAppStore((state) =>
    state.instanceMissingLicense(resolvedFeature, instance)
  );
  const requiredPlan = useAppStore((state) =>
    state.getMinimumRequiredPlan(resolvedFeature)
  );

  if (!feature) {
    return null;
  }

  const featureKey = PlanFeature[feature].split(".").join("-");
  const title = instanceMissingLicense
    ? t("subscription.instance-assignment.require-license")
    : t("subscription.disabled-feature");

  const close = () => onOpenChange(false);

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

  const requiredPlanLabel = t(
    `subscription.plan.${planLabel[requiredPlan] ?? "enterprise"}.title`
  );
  const startTrialLabel = hasPermission
    ? t("subscription.trial-for-days", { days: trialingDays })
    : t("subscription.contact-to-upgrade");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <div className="p-6">
          <div className="flex items-center justify-between border-b pb-2 mb-4">
            <DialogTitle className="text-base font-medium">{title}</DialogTitle>
            {/* Vue's BBModal had a built-in X close affordance; the React
                Dialog primitive does not, so render one explicitly so
                users can dismiss the paywall regardless of which CTA path
                renders below. */}
            <DialogClose
              aria-label={t("common.close")}
              className="rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent cursor-pointer"
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
            ) : showTrial && !instanceMissingLicense ? (
              <Button
                variant="default"
                onClick={() => {
                  window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
                  // Vue's BBModal default-closes after a CTA click; mirror
                  // that so the paywall doesn't linger after the inquiry
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
