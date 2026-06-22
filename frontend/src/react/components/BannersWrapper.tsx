import { AlertCircle, ShoppingCart, Sparkles, Wrench, X } from "lucide-react";
import { useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { AnnouncementBanner } from "@/react/components/AnnouncementBanner";
import { resolveAnnouncementTheme } from "@/react/components/announcement-theme";
import { RouterLink } from "@/react/components/RouterLink";
import { Button, buttonVariants } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  usePlanFeature,
  useServerState,
  useSubscriptionState,
  useWorkspacePermission,
  useWorkspaceProfile,
} from "@/react/hooks/useAppState";
import {
  SETTING_ROUTE_WORKSPACE_GENERAL,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
} from "@/react/router";
import { useAppStore } from "@/react/stores/app";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { isDev, urlfy } from "@/utils";

const LICENSE_EXPIRATION_THRESHOLD = 7;

const planLabelKey: Partial<Record<PlanType, string>> = {
  [PlanType.FREE]: "free",
  [PlanType.TEAM]: "team",
  [PlanType.ENTERPRISE]: "enterprise",
};

function planTitleKey(plan: PlanType) {
  return `subscription.plan.${planLabelKey[plan] ?? "enterprise"}.title`;
}

function planFeatureFromKey(key: string): PlanFeature {
  const feature = PlanFeature[key as keyof typeof PlanFeature];
  return typeof feature === "number"
    ? feature
    : PlanFeature.FEATURE_UNSPECIFIED;
}

function BannerDismissButton({
  onClick,
  label,
}: {
  readonly onClick: () => void;
  readonly label: string;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      className="text-white hover:bg-white/10 hover:text-white"
      onClick={onClick}
    >
      <span className="sr-only">{label}</span>
      <X className="size-5" />
    </Button>
  );
}

function BannerExternalUrl() {
  const { t } = useTranslation();
  const hasPermission = useWorkspacePermission(
    "bb.settings.setWorkspaceProfile"
  );
  const [show, setShow] = useState(true);

  if (!show) return null;

  return (
    <div className="bg-accent">
      <div className="mx-auto px-3 py-3">
        <div className="flex flex-wrap items-center justify-between">
          <div className="flex w-0 flex-1 items-center">
            <p className="ml-3 truncate font-medium text-white">
              {t("banner.external-url")}
            </p>
          </div>
          {hasPermission ? (
            <div className="order-3 mt-2 w-full shrink-0 sm:order-2 sm:mt-0 sm:w-auto">
              <RouterLink
                to={{ name: SETTING_ROUTE_WORKSPACE_GENERAL }}
                className={buttonVariants({
                  className:
                    "h-auto rounded-md bg-white py-2 pr-2 pl-4 text-base font-medium text-accent shadow-xs hover:bg-indigo-50",
                })}
              >
                {t("common.configure-now")}
                <Wrench className="ml-1 size-5" />
              </RouterLink>
            </div>
          ) : null}
          <div className="order-2 -mr-1 shrink-0 sm:order-3 sm:ml-3">
            <BannerDismissButton
              label={t("common.dismiss")}
              onClick={() => setShow(false)}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function BannerSubscription() {
  const { t } = useTranslation();
  const { currentPlan, daysBeforeExpire, expireAt, isExpired, isTrialing } =
    useSubscriptionState();

  const content = useMemo(() => {
    if (currentPlan === PlanType.FREE) return "";

    const plan = t(planTitleKey(currentPlan));
    if (isTrialing) {
      return t("banner.trial-expires", {
        days: daysBeforeExpire,
        expireAt,
        plan,
      });
    }
    if (isExpired) {
      return t("banner.license-expired", { expireAt, plan });
    }
    if (daysBeforeExpire <= LICENSE_EXPIRATION_THRESHOLD) {
      return t("banner.license-expires", {
        days: daysBeforeExpire,
        expireAt,
        plan,
      });
    }
    return "";
  }, [currentPlan, daysBeforeExpire, expireAt, isExpired, isTrialing, t]);

  if (!content) return null;

  return (
    <div className="bg-info">
      <div className="mx-auto px-3 py-1">
        <div className="flex flex-wrap items-center justify-center gap-x-2">
          <p className="ml-3 truncate text-base font-medium text-white">
            {content}
          </p>
          <RouterLink
            to={{ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION }}
            className="flex cursor-pointer items-center justify-center py-1 text-base font-medium text-white underline hover:opacity-80"
          >
            {t(
              isTrialing
                ? "subscription.purchase.subscribe"
                : "subscription.purchase.update"
            )}
            <ShoppingCart className="ml-1 size-5 text-white" />
          </RouterLink>
        </div>
      </div>
    </div>
  );
}

function BannerAnnouncement() {
  const workspaceProfile = useWorkspaceProfile();
  const hasAnnouncementFeature = usePlanFeature(
    PlanFeature.FEATURE_DASHBOARD_ANNOUNCEMENT
  );
  const announcement = workspaceProfile?.announcement;
  const text = announcement?.text ?? "";
  const rawLink = announcement?.link ?? "";
  const link = rawLink ? urlfy(rawLink) : "";
  const { background, text: textColor } =
    resolveAnnouncementTheme(announcement);

  if (!text || !hasAnnouncementFeature) return null;

  return (
    <AnnouncementBanner
      text={text}
      link={link}
      background={background}
      textColor={textColor}
    />
  );
}

function BannerUpgradeSubscription() {
  const { t } = useTranslation();
  const { serverInfo } = useServerState();
  const { currentPlan } = useSubscriptionState();
  const getMinimumRequiredPlan = useAppStore(
    (state) => state.getMinimumRequiredPlan
  );
  const [showModal, setShowModal] = useState(false);
  const unlicensedFeatures = serverInfo?.unlicensedFeatures ?? [];
  const neededPlan = useMemo(() => {
    let plan = PlanType.FREE;
    for (const featureKey of unlicensedFeatures) {
      const requiredPlan = getMinimumRequiredPlan(
        planFeatureFromKey(featureKey)
      );
      if (requiredPlan > plan) {
        plan = requiredPlan;
      }
    }
    return plan;
  }, [getMinimumRequiredPlan, unlicensedFeatures]);
  const showBanner = unlicensedFeatures.length > 0 && neededPlan > currentPlan;
  const currentPlanTitle = t(planTitleKey(currentPlan));
  const neededPlanTitle = t(planTitleKey(neededPlan));
  const neededPlanFeatures = t("subscription.plan-features", {
    plan: neededPlanTitle,
  });

  const gotoSubscriptionPage = () => {
    setShowModal(false);
  };

  if (!showBanner) return null;

  return (
    <>
      <div className="overflow-clip bg-gray-200">
        <div className="bb-banner-scroll h-10 w-full">
          <div className="mx-auto flex w-full flex-row flex-wrap items-center justify-center px-3 py-1">
            <div className="flex flex-row items-center">
              <AlertCircle className="mr-1 h-auto w-5 text-gray-800" />
              <Trans
                t={t}
                i18nKey="subscription.overuse-warning"
                values={{
                  currentPlan: currentPlanTitle,
                  neededPlan: neededPlanFeatures,
                }}
                components={{
                  neededPlan: (
                    <button
                      type="button"
                      className="mr-1 cursor-pointer underline hover:opacity-60"
                      onClick={() => setShowModal(true)}
                    >
                      {neededPlanFeatures}
                    </button>
                  ),
                }}
              />
            </div>
            <div className="ml-2">
              <RouterLink
                to={{ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION }}
                className={buttonVariants({ size: "sm" })}
                onClick={gotoSubscriptionPage}
              >
                <Sparkles className="h-auto w-4" />
                {t("subscription.upgrade")}
              </RouterLink>
            </div>
          </div>
        </div>
      </div>

      <Dialog open={showModal} onOpenChange={setShowModal}>
        <DialogContent className="max-w-lg">
          <div>
            <DialogTitle className="mb-4 text-base font-medium">
              {t("subscription.upgrade-now")}?
            </DialogTitle>
            <p>
              {t("subscription.overuse-modal.description", {
                plan: currentPlanTitle,
              })}
            </p>
            <div className="my-2 pl-4">
              <ul className="list-inside list-disc">
                {unlicensedFeatures.map((featureKey) => {
                  const feature = planFeatureFromKey(featureKey);
                  const requiredPlan = getMinimumRequiredPlan(feature);
                  return (
                    <li key={featureKey}>
                      {t(`dynamic.subscription.features.${featureKey}.title`)} (
                      {t(planTitleKey(requiredPlan))})
                    </li>
                  );
                })}
              </ul>
            </div>
            <div className="mt-3 mb-4 w-full">
              <RouterLink
                to={{ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION }}
                className={buttonVariants({ className: "w-full" })}
                onClick={gotoSubscriptionPage}
              >
                {t("subscription.upgrade-now")}
              </RouterLink>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function BannersWrapper() {
  const { needConfigureExternalUrl } = useServerState();
  const { currentPlan, daysBeforeExpire, isExpired, isTrialing } =
    useSubscriptionState();

  const shouldShowSubscriptionBanner =
    isExpired ||
    isTrialing ||
    (currentPlan !== PlanType.FREE &&
      daysBeforeExpire <= LICENSE_EXPIRATION_THRESHOLD);
  const shouldShowExternalUrlBanner = !isDev() && needConfigureExternalUrl;

  return (
    <>
      <BannerUpgradeSubscription />
      {shouldShowSubscriptionBanner ? <BannerSubscription /> : null}
      {shouldShowExternalUrlBanner ? <BannerExternalUrl /> : null}
      <BannerAnnouncement />
    </>
  );
}
