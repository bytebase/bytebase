import { Check, Loader2, Minus, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { pushNotification, useSubscriptionV1Store } from "@/store";
import type { PurchasePlanAdditional } from "@/types/proto-es/v1/subscription_service_pb";
import {
  BillingInterval,
  PlanType,
  PurchaseDiscount_Type,
  PurchasePlanAdditional_Type,
} from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

interface PurchaseSectionProps {
  onRequireEnterprise: () => void;
}

// Data model matching the Vue PlanCardData interface.
interface PlanCardData {
  type: PlanType;
  title: string;
  description: string;
  features: { text: string; bold?: boolean }[];
  selfServicePurchase: boolean;
  unitPrice: number;
  userAdditional: PurchasePlanAdditional | undefined;
  billingIntervals: BillingInterval[];
  discountDescription: string;
}

// Feature keys for i18n (display-only, not from API).
const featureKeys: Record<number, string[]> = {
  [PlanType.FREE]: [
    "limits",
    "dcm",
    "git",
    "sql-review",
    "backup",
    "sql-editor",
    "schema",
    "iam",
    "support",
  ],
  [PlanType.TEAM]: [
    "limits",
    "everything",
    "batch-query",
    "read-only",
    "query-policy",
    "sso",
    "groups",
    "db-groups",
    "audit-log",
    "support",
  ],
  [PlanType.ENTERPRISE]: [
    "limits",
    "everything",
    "custom-limits",
    "approval",
    "audit-log",
    "sso",
    "2fa",
    "masking",
    "roles",
    "scim",
    "secret",
    "branding",
    "support",
  ],
};

const planPrefix: Record<number, string> = {
  [PlanType.FREE]: "free",
  [PlanType.TEAM]: "pro",
  [PlanType.ENTERPRISE]: "enterprise",
};

export function PurchaseSection({ onRequireEnterprise }: PurchaseSectionProps) {
  const { t } = useTranslation();
  const subscriptionStore = useSubscriptionV1Store();

  const currentPlan = useVueState(() => subscriptionStore.currentPlan);
  const isFreePlan = useVueState(() => subscriptionStore.isFreePlan);
  const isExpired = useVueState(() => subscriptionStore.isExpired);
  const subscription = useVueState(() => subscriptionStore.subscription);
  const paymentInfo = useVueState(() => subscriptionStore.paymentInfo);
  const purchasePlans = useVueState(() => subscriptionStore.purchasePlans);

  const allowManage = hasWorkspacePermissionV2("bb.subscription.manage");

  const [seats, setSeats] = useState(1);
  const [loading, setLoading] = useState(false);
  const [canceling, setCanceling] = useState(false);
  const [checkPolicy, setCheckPolicy] = useState(false);
  const [pendingPayment, setPendingPayment] = useState(false);

  // Sync state from subscription when it changes.
  useEffect(() => {
    if (subscription && currentPlan !== PlanType.FREE && !isExpired) {
      setSeats(Math.max(1, subscription.seats));
      setCheckPolicy(true);
    }
  }, [subscription, currentPlan, isExpired]);

  // Fetch plans on mount.
  useEffect(() => {
    subscriptionStore.fetchPurchasePlans();
  }, []);

  // Handle session_id polling on mount.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const sessionId = params.get("session_id");
    if (!sessionId || !isFreePlan || !allowManage) return;

    let cancelled = false;
    (async () => {
      try {
        const status = await subscriptionStore.verifyCheckoutSession(sessionId);
        if (status !== "complete" || cancelled) return;

        setPendingPayment(true);
        for (let i = 0; i < 30; i++) {
          if (cancelled) break;
          await subscriptionStore.fetchSubscription();
          if (!subscriptionStore.isFreePlan) break;
          await new Promise((r) => setTimeout(r, 2000));
        }
        setPendingPayment(false);
      } catch (e) {
        console.error("failed to verify checkout session", e);
      }
      // Clean up query params.
      window.history.replaceState({}, "", window.location.pathname);
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  // Fetch payment info for active subscriptions.
  useEffect(() => {
    if (!isFreePlan && !isExpired) {
      subscriptionStore.fetchPaymentInfo();
    }
  }, [isFreePlan, isExpired]);

  const isCurrentPlan = useCallback(
    (plan: PlanType) => currentPlan === plan && !isExpired,
    [currentPlan, isExpired]
  );

  // Helpers matching Vue computed functions.
  const planTitle = (type: PlanType): string => {
    switch (type) {
      case PlanType.FREE:
        return t("subscription.plan.free.title");
      case PlanType.TEAM:
        return t("subscription.plan.team.title");
      case PlanType.ENTERPRISE:
        return t("subscription.plan.enterprise.title");
      default:
        return "";
    }
  };

  const planDescription = (type: PlanType): string => {
    switch (type) {
      case PlanType.FREE:
        return t("subscription.purchase.free-description");
      case PlanType.TEAM:
        return t("subscription.purchase.pro-description");
      case PlanType.ENTERPRISE:
        return t("subscription.purchase.enterprise-description");
      default:
        return "";
    }
  };

  const getFeaturesForPlan = (type: PlanType) =>
    (featureKeys[type] || []).map((key) => ({
      text: t(
        `dynamic.subscription.purchase.features.${planPrefix[type]}.${key}`
      ),
      bold: key === "everything",
    }));

  // Build planCards matching the Vue computed.
  const planCards = useMemo((): PlanCardData[] => {
    if (purchasePlans.length === 0) return [];

    const cards: PlanCardData[] = [
      {
        type: PlanType.FREE,
        title: planTitle(PlanType.FREE),
        description: planDescription(PlanType.FREE),
        features: getFeaturesForPlan(PlanType.FREE),
        selfServicePurchase: false,
        unitPrice: 0,
        userAdditional: undefined,
        billingIntervals: [],
        discountDescription: "",
      },
    ];

    for (const plan of purchasePlans) {
      const userAdditional = plan.additionals.find(
        (a) => a.type === PurchasePlanAdditional_Type.USER
      );
      const discount = plan.billingMethods.find((bm) => bm.discount)?.discount;
      let discountDescription = "";
      if (discount) {
        switch (discount.type) {
          case PurchaseDiscount_Type.PERCENTAGE_OFF:
            discountDescription = t(
              "subscription.purchase.discount.percentage-off",
              { value: discount.value }
            );
            break;
          case PurchaseDiscount_Type.FIXED_MONTH_OFF:
            discountDescription = t(
              "subscription.purchase.discount.month-off",
              { value: discount.value }
            );
            break;
          case PurchaseDiscount_Type.FIXED_PRICE_OFF:
            discountDescription = t(
              "subscription.purchase.discount.price-off",
              { value: discount.value }
            );
            break;
          default:
            break;
        }
      }

      cards.push({
        type: plan.type,
        title: planTitle(plan.type),
        description: planDescription(plan.type),
        features: getFeaturesForPlan(plan.type),
        selfServicePurchase: plan.selfServicePurchase,
        unitPrice: userAdditional?.unitPrice ?? 0,
        userAdditional,
        billingIntervals: plan.billingMethods.map((bm) => bm.interval),
        discountDescription,
      });
    }

    return cards;
  }, [purchasePlans, t]);

  const isPlanConfigChanged = (card: PlanCardData): boolean => {
    if (!isCurrentPlan(card.type)) return false;
    return seats !== (subscription?.seats ?? 0);
  };

  const purchaseButtonDisabled = (card: PlanCardData): boolean => {
    if (loading) return true;
    if (!checkPolicy) return true;
    if (isCurrentPlan(card.type) && !isPlanConfigChanged(card)) return true;
    return false;
  };

  const purchaseButtonText = (card: PlanCardData): string => {
    if (isPlanConfigChanged(card)) return t("subscription.purchase.update");
    if (isCurrentPlan(card.type)) return t("subscription.current");
    const label = t("subscription.purchase.subscribe");
    return card.discountDescription
      ? `${label} (${card.discountDescription})`
      : label;
  };

  const handlePurchase = async (card: PlanCardData) => {
    if (purchaseButtonDisabled(card)) return;

    const interval = card.billingIntervals[0] ?? BillingInterval.MONTH;
    setLoading(true);
    try {
      let paymentUrl: string;
      if (isPlanConfigChanged(card)) {
        paymentUrl = await subscriptionStore.updatePurchase(
          card.type,
          interval,
          seats,
          subscription?.etag ?? ""
        );
      } else {
        paymentUrl = await subscriptionStore.createPurchase(
          card.type,
          interval,
          seats
        );
      }
      if (paymentUrl) {
        window.location.href = paymentUrl;
      } else {
        // Direct update — poll without updating store to avoid UI flashing.
        setPendingPayment(true);
        for (let i = 0; i < 30; i++) {
          const sub = await subscriptionStore.fetchSubscription(false);
          if (sub && sub.plan !== PlanType.FREE && sub.seats === seats) {
            subscriptionStore.setSubscription(sub);
            break;
          }
          await new Promise((r) => setTimeout(r, 2000));
        }
        setPendingPayment(false);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = async () => {
    if (!confirm(t("subscription.purchase.cancel-confirm"))) return;
    setCanceling(true);
    try {
      await subscriptionStore.cancelPurchase();
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("subscription.purchase.canceled"),
      });
    } catch (e) {
      console.error(e);
    } finally {
      setCanceling(false);
    }
  };

  // Pending payment overlay.
  if (pendingPayment) {
    return (
      <div className="flex flex-col items-center justify-center py-16 gap-y-4">
        <Loader2 className="h-8 w-8 animate-spin text-accent" />
        <div className="text-lg font-medium">
          {t("subscription.purchase.pending-title")}
        </div>
        <div className="text-sm text-control-placeholder">
          {t("subscription.purchase.pending-description")}
        </div>
      </div>
    );
  }

  return (
    <div className="w-full">
      {/* Active Subscription Management */}
      {!isFreePlan && !isExpired && (
        <>
          {paymentInfo && (
            <div className="space-y-2">
              <div className="text-lg font-medium">
                {t("subscription.purchase.payment-info")}
              </div>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-control-placeholder">
                    {t("subscription.purchase.total-price")}
                  </span>
                  <div className="flex items-center gap-x-2">
                    <div className="font-medium">
                      ${(Number(paymentInfo.totalPrice) / 100).toFixed(2)}{" "}
                      {paymentInfo.currency.toUpperCase()}
                    </div>
                    {paymentInfo.invoiceUrl && (
                      <a
                        href={paymentInfo.invoiceUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-accent hover:underline text-sm"
                      >
                        {t("subscription.purchase.manage-invoices")}
                      </a>
                    )}
                  </div>
                </div>
                <div>
                  <span className="text-control-placeholder">
                    {t("subscription.purchase.billing-period")}
                  </span>
                  <div className="font-medium">
                    {paymentInfo.periodStart} - {paymentInfo.periodEnd}
                  </div>
                </div>
              </div>
            </div>
          )}
          {paymentInfo?.cancelAtPeriodEnd ? (
            <Alert variant="warning" className="my-2">
              {t("subscription.purchase.cancel-pending", {
                date: paymentInfo.periodEnd,
              })}
            </Alert>
          ) : (
            allowManage && (
              <div className="mt-4">
                <Button
                  variant="destructive"
                  disabled={canceling}
                  onClick={handleCancel}
                >
                  {canceling && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  {t("subscription.purchase.cancel")}
                </Button>
              </div>
            )
          )}
          <hr className="my-6 border-t border-block-border" />
        </>
      )}

      {/* Plan Cards */}
      {planCards.length > 0 && (
        <>
          <div className="grid gap-8 grid-cols-1 lg:grid-cols-3">
            {planCards.map((card) => (
              <PlanCard
                key={card.type}
                title={card.title}
                description={card.description}
                features={card.features}
                highlighted={isCurrentPlan(card.type)}
                pricing={
                  card.type === PlanType.FREE ? (
                    <span className="text-4xl font-extrabold">$0</span>
                  ) : card.unitPrice > 0 ? (
                    <>
                      <span className="text-4xl font-extrabold">
                        ${card.unitPrice / 100}
                      </span>
                      <span className="ml-2 text-control-placeholder">
                        {t("subscription.purchase.per-user-per-month")}
                      </span>
                    </>
                  ) : (
                    <span className="text-3xl font-extrabold">
                      {t("subscription.purchase.custom")}
                    </span>
                  )
                }
                config={
                  allowManage && card.userAdditional ? (
                    <div className="flex items-center">
                      <span className="text-main">
                        {t("subscription.purchase.seats")}
                      </span>
                      <div className="ml-auto flex items-center h-7 rounded-lg bg-gray-200 text-gray-600">
                        <button
                          type="button"
                          className="px-2 h-full rounded-l hover:bg-gray-300 disabled:opacity-50 cursor-pointer"
                          disabled={
                            seats <= (card.userAdditional.minimumCount || 1)
                          }
                          onClick={() =>
                            setSeats(
                              Math.max(
                                card.userAdditional!.minimumCount || 1,
                                seats - 1
                              )
                            )
                          }
                        >
                          <Minus className="h-3 w-3" />
                        </button>
                        <span className="w-8 text-center text-sm">{seats}</span>
                        <button
                          type="button"
                          className="px-2 h-full rounded-r hover:bg-gray-300 disabled:opacity-50 cursor-pointer"
                          disabled={
                            card.userAdditional.maximumCount > 0 &&
                            seats >= card.userAdditional.maximumCount
                          }
                          onClick={() => setSeats(seats + 1)}
                        >
                          <Plus className="h-3 w-3" />
                        </button>
                      </div>
                    </div>
                  ) : undefined
                }
                action={
                  allowManage ? (
                    card.type === PlanType.FREE ? (
                      isCurrentPlan(PlanType.FREE) ? (
                        <Button
                          variant="outline"
                          className="w-full"
                          disabled={isCurrentPlan(PlanType.FREE)}
                        >
                          {t("subscription.current")}
                        </Button>
                      ) : undefined
                    ) : card.selfServicePurchase ? (
                      <>
                        <label className="mt-3 flex items-start gap-x-2 text-sm text-control-placeholder cursor-pointer">
                          <input
                            type="checkbox"
                            checked={checkPolicy}
                            onChange={(e) => setCheckPolicy(e.target.checked)}
                            className="mt-0.5"
                          />
                          <span>
                            {t("subscription.purchase.accept-terms-prefix")}{" "}
                            <a
                              href="https://www.bytebase.com/terms"
                              target="_blank"
                              rel="noopener noreferrer"
                              className="underline hover:text-main"
                            >
                              {t("subscription.purchase.terms-of-service")}
                            </a>{" "}
                            {t("subscription.purchase.and")}{" "}
                            <a
                              href="https://www.bytebase.com/privacy"
                              target="_blank"
                              rel="noopener noreferrer"
                              className="underline hover:text-main"
                            >
                              {t("subscription.purchase.privacy-policy")}
                            </a>
                          </span>
                        </label>
                        <Button
                          className="mt-3 w-full"
                          disabled={purchaseButtonDisabled(card)}
                          onClick={() => handlePurchase(card)}
                        >
                          {loading && (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          )}
                          {purchaseButtonText(card)}
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="outline"
                        className="w-full"
                        onClick={onRequireEnterprise}
                      >
                        {t("subscription.contact-us")}
                      </Button>
                    )
                  ) : undefined
                }
              />
            ))}
          </div>

          {/* Footer */}
          <div className="pt-4 pb-2 text-center">
            <a
              className="text-sm text-control-placeholder hover:text-main underline"
              href="https://www.bytebase.com/pricing?source=console"
              target="_blank"
              rel="noopener noreferrer"
            >
              {t("subscription.purchase.see-comparison")}
            </a>
          </div>
        </>
      )}
    </div>
  );
}

// Plan card component matching Vue's PlanCard.vue.
function PlanCard({
  title,
  description,
  features,
  highlighted,
  pricing,
  config,
  action,
}: {
  title: string;
  description: string;
  features: { text: string; bold?: boolean }[];
  highlighted: boolean;
  pricing: React.ReactNode;
  config?: React.ReactNode;
  action?: React.ReactNode;
}) {
  return (
    <div
      className={`h-full flex flex-col rounded-lg p-5 border-2 ${highlighted ? "shadow-lg border-accent" : "border-block-border"}`}
    >
      <h3 className="text-3xl font-semibold text-main">{title}</h3>
      <p className="text-control-placeholder mt-1 text-sm">{description}</p>
      <div className="mt-4 mb-2 flex items-baseline">{pricing}</div>
      <div className="mb-4 space-y-1 text-sm">
        {features.map((f, i) => (
          <div key={i} className="flex items-start">
            <Check className="h-3.5 w-3.5 text-control-light mr-2 mt-0.5 shrink-0" />
            <span
              className={
                f.bold
                  ? "font-semibold text-main leading-5"
                  : "text-main leading-5"
              }
            >
              {f.text}
            </span>
          </div>
        ))}
      </div>
      <div className="flex-1">{config}</div>
      {action && <div className="mt-3">{action}</div>}
    </div>
  );
}
