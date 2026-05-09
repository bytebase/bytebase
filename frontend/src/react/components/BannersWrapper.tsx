import {
  AlertCircle,
  ArrowRight,
  Cloud,
  Download,
  ShoppingCart,
  Sparkles,
  Wrench,
  X,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
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
  useNavigate,
} from "@/react/router";
import { useAppStore } from "@/react/stores/app";
import {
  type Announcement,
  Announcement_AlertLevel,
} from "@/types/proto-es/v1/setting_service_pb";
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

function BannerDemo() {
  const { t } = useTranslation();
  const [show, setShow] = useState(true);

  if (!show) return null;

  return (
    <div className="bg-accent">
      <div className="mx-auto px-3 py-1">
        <div className="flex flex-wrap items-center justify-between">
          <div className="flex w-0 flex-1 items-center">
            <p className="flex items-center truncate font-medium text-white">
              <a
                href="https://cal.com/bytebase/product-walkthrough"
                target="_blank"
                rel="noreferrer"
                className="flex items-center gap-x-1 underline"
              >
                {t("banner.request-demo")}
              </a>
            </p>
          </div>
          <div className="order-3 my-2 flex w-full shrink-0 flex-row gap-x-4 sm:order-2 sm:my-0 sm:w-auto sm:py-1">
            <a
              href="https://docs.bytebase.com/get-started/self-host-vs-cloud/?source=demo"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-center rounded-xs border border-transparent bg-white py-1 pr-2 pl-4 text-base font-medium text-accent hover:bg-indigo-50"
            >
              {t("banner.deploy")}
              <Download className="ml-1 size-5" />
            </a>
            <a
              href="https://hub.bytebase.com/workspace?source=demo"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-center rounded-xs border border-transparent bg-white py-1 pr-2 pl-4 text-base font-medium text-accent hover:bg-indigo-50"
            >
              {t("banner.cloud")}
              <Cloud className="ml-1 size-5" />
            </a>
          </div>
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

function BannerExternalUrl() {
  const { t } = useTranslation();
  const navigate = useNavigate();
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
              <Button
                type="button"
                className="h-auto rounded-md bg-white py-2 pr-2 pl-4 text-base font-medium text-accent shadow-xs hover:bg-indigo-50"
                onClick={() => {
                  void navigate.push({ name: SETTING_ROUTE_WORKSPACE_GENERAL });
                }}
              >
                {t("common.configure-now")}
                <Wrench className="ml-1 size-5" />
              </Button>
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
  const navigate = useNavigate();
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
          <button
            type="button"
            className="flex cursor-pointer items-center justify-center py-1 text-base font-medium text-white underline hover:opacity-80"
            onClick={() => {
              void navigate.push({
                name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
              });
            }}
          >
            {t(
              isTrialing
                ? "subscription.purchase.subscribe"
                : "subscription.purchase.update"
            )}
            <ShoppingCart className="ml-1 size-5 text-white" />
          </button>
        </div>
      </div>
    </div>
  );
}

function announcementColors(level: Announcement["level"] | undefined) {
  switch (level) {
    case Announcement_AlertLevel.WARNING:
      return "bg-warning hover:bg-warning-hover";
    case Announcement_AlertLevel.CRITICAL:
      return "bg-error hover:bg-error-hover";
    default:
      return "bg-info hover:bg-info-hover";
  }
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

  if (!text || !hasAnnouncementFeature) return null;

  return (
    <div
      className={`mx-auto flex w-full flex-row flex-wrap justify-center px-3 py-1 text-center font-medium text-white ${announcementColors(
        announcement?.level
      )}`}
    >
      {link ? (
        <a
          href={link}
          target="_blank"
          rel="noreferrer"
          className="flex flex-row items-center hover:underline"
        >
          <p className="px-1">{text}</p>
          <ArrowRight className="mr-3 size-5" />
        </a>
      ) : (
        <p>{text}</p>
      )}
    </div>
  );
}

function BannerUpgradeSubscription() {
  const { t } = useTranslation();
  const navigate = useNavigate();
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
    void navigate.push({ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION });
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
                      className="cursor-pointer underline hover:opacity-60"
                      onClick={() => setShowModal(true)}
                    >
                      {neededPlanFeatures}
                    </button>
                  ),
                }}
              />
            </div>
            <div className="ml-2">
              <Button size="sm" onClick={gotoSubscriptionPage}>
                {t("subscription.upgrade")}
                <Sparkles className="ml-1 h-auto w-4 text-accent" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <Dialog open={showModal} onOpenChange={setShowModal}>
        <DialogContent className="max-w-lg">
          <div className="p-6">
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
              <Button className="w-full" onClick={gotoSubscriptionPage}>
                {t("subscription.upgrade-now")}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function BannersWrapper() {
  const { serverInfo, needConfigureExternalUrl } = useServerState();
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
      {serverInfo?.demo ? <BannerDemo /> : null}
      {shouldShowSubscriptionBanner ? <BannerSubscription /> : null}
      {shouldShowExternalUrlBanner ? <BannerExternalUrl /> : null}
      <BannerAnnouncement />
    </>
  );
}
