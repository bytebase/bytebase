import { Lock, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { useSubscriptionV1Store } from "@/store";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { Tooltip } from "./ui/tooltip";

interface FeatureBadgeProps {
  readonly feature: PlanFeature;
  readonly instance?: Instance | InstanceResource;
  readonly clickable?: boolean;
  readonly className?: string;
}

const planLabel: Record<number, string> = {
  [PlanType.FREE]: "free",
  [PlanType.TEAM]: "team",
  [PlanType.ENTERPRISE]: "enterprise",
};

/**
 * FeatureBadge shows a sparkles icon with tooltip when the user's plan
 * doesn't include the required feature, or a lock icon when the instance
 * is missing a license assignment.
 *
 * Renders nothing if the feature is available.
 */
export function FeatureBadge({
  feature,
  instance,
  clickable = true,
  className,
}: FeatureBadgeProps) {
  const { t } = useTranslation();
  const subscriptionStore = useSubscriptionV1Store();

  const hasFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(feature, instance)
  );
  const instanceMissingLicense = useVueState(() =>
    subscriptionStore.instanceMissingLicense(feature, instance)
  );
  const minimumPlan = useVueState(() =>
    subscriptionStore.getMinimumRequiredPlan(feature)
  );

  if (instanceMissingLicense) {
    return (
      <Tooltip
        content={t(
          "subscription.instance-assignment.missing-license-attention"
        )}
      >
        <span className={className ?? "text-accent inline-flex"}>
          <Lock className="w-5 h-5" />
        </span>
      </Tooltip>
    );
  }

  if (hasFeature) {
    return null;
  }

  const requiredPlanLabel = t(
    `subscription.plan.${planLabel[minimumPlan] ?? "enterprise"}.title`
  );
  const tooltip = t("subscription.require-subscription", {
    requiredPlan: requiredPlanLabel,
  });

  if (clickable) {
    return (
      <Tooltip content={tooltip}>
        <a
          href="/setting/subscription"
          className={className ?? "text-accent inline-flex"}
        >
          <Sparkles className="w-5 h-5" />
        </a>
      </Tooltip>
    );
  }

  return (
    <Tooltip content={tooltip}>
      <span className={className ?? "text-accent inline-flex"}>
        <Sparkles className="w-5 h-5" />
      </span>
    </Tooltip>
  );
}
