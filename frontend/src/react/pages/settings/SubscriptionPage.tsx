import { Copy, Pencil } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { SubscriptionData } from "@/react/types";

interface SubscriptionPageProps {
  data: SubscriptionData;
  allowEdit: boolean;
  allowManageInstanceLicenses: boolean;
  onUploadLicense: (license: string) => Promise<boolean>;
  onRequireEnterprise: () => void;
  onManageInstanceLicenses: () => void;
}

export function SubscriptionPage({
  data,
  allowEdit,
  allowManageInstanceLicenses,
  onUploadLicense,
  onRequireEnterprise,
  onManageInstanceLicenses,
}: SubscriptionPageProps) {
  const { t } = useTranslation();
  const [license, setLicense] = useState("");
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  const disabled = loading || !license;

  const userLimit =
    data.userCountLimit === Number.MAX_VALUE
      ? t("common.unlimited")
      : `${data.userCountLimit}`;

  const totalLicenseCount =
    data.instanceLicenseCount === Number.MAX_VALUE
      ? t("common.unlimited")
      : `${data.instanceLicenseCount}`;

  const handleUpload = async () => {
    if (disabled) return;
    setLoading(true);
    try {
      const success = await onUploadLicense(license);
      if (success) {
        setLicense("");
      }
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(data.workspaceId);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API not available
    }
  };

  return (
    <div className="w-full px-4 py-4 mx-auto">
      {/* Plan stats grid */}
      <div className="w-full grid grid-cols-2 gap-6 lg:grid-cols-3 3xl:grid-cols-4 my-4">
        {/* Current plan */}
        <div className="flex flex-col text-left">
          <div className="flex items-center text-main">
            {t("subscription.current")}
            {data.isExpired && (
              <span className="ml-2 inline-flex items-center px-3 py-0.5 rounded-full text-base font-sm bg-red-100 text-red-800 h-6">
                {t("subscription.expired")}
              </span>
            )}
            {!data.isExpired && data.isTrialing && (
              <span className="ml-2 inline-flex items-center px-3 py-0.5 rounded-full text-base font-sm bg-indigo-100 text-indigo-800 h-6">
                {t("subscription.trialing")}
              </span>
            )}
          </div>
          <div className="text-indigo-600 mt-1 text-3xl lg:text-4xl">
            {data.currentPlan}
          </div>
        </div>

        {/* Expires at */}
        {!data.isFreePlan && (
          <div className="flex flex-col text-left">
            <div className="text-main">{t("subscription.expires-at")}</div>
            <dd className="mt-1 text-3xl lg:text-4xl">
              {data.expireAt || "N/A"}
            </dd>
          </div>
        )}

        {/* Free trial */}
        {data.showTrial && allowEdit && (
          <div className="flex flex-col text-left">
            <div className="text-main">{t("subscription.try-for-free")}</div>
            <div className="mt-1">
              <button
                className="rounded-md bg-indigo-600 px-4 py-2 text-white text-lg hover:bg-indigo-700"
                onClick={onRequireEnterprise}
              >
                {t("subscription.enterprise-free-trial", {
                  days: data.trialingDays,
                })}
              </button>
            </div>
          </div>
        )}

        {/* Inquire enterprise */}
        {data.isTrialing && data.planType === "ENTERPRISE" && (
          <div className="flex flex-col text-left">
            <div className="text-main">
              {t("subscription.inquire-enterprise-plan")}
            </div>
            <div className="mt-1 ml-auto">
              <button
                className="rounded-md bg-indigo-600 px-4 py-2 text-white text-lg hover:bg-indigo-700"
                onClick={onRequireEnterprise}
              >
                {t("subscription.contact-us")}
              </button>
            </div>
          </div>
        )}

        {/* Instance license stats */}
        {allowManageInstanceLicenses && (
          <InstanceLicenseStats
            planType={data.planType}
            instanceCountLimit={data.instanceCountLimit}
            activatedCount={data.activatedInstanceCount}
            totalLicenseCount={totalLicenseCount}
            onManageInstanceLicenses={onManageInstanceLicenses}
          />
        )}

        {/* User count */}
        <div className="flex flex-col text-left">
          <div className="text-main">
            {t("subscription.instance-assignment.used-and-total-user")}
          </div>
          <div className="mt-1 text-4xl flex items-center gap-2">
            {data.activeUserCount}
            <span className="font-mono text-gray-500">/</span>
            {userLimit}
          </div>
        </div>
      </div>

      {/* Divider */}
      <hr className="my-6 border-t border-gray-200" />

      {/* Workspace ID */}
      <div>
        <label className="flex items-center gap-x-2">
          <span className="text-main">
            {t("settings.general.workspace.id")}
          </span>
        </label>
        <div className="mb-3 text-sm text-gray-400">
          {t("settings.general.workspace.id-description")}
        </div>
        <div className="mb-4 flex items-center gap-x-2">
          <input
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm bg-gray-50 cursor-default focus:outline-none"
            readOnly
            value={data.workspaceId}
            onClick={(e) => (e.target as HTMLInputElement).select()}
          />
          <button
            className="inline-flex items-center justify-center rounded-md p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100"
            onClick={handleCopy}
            title={t("common.copy")}
          >
            <Copy className="h-4 w-4" />
          </button>
          {copied && (
            <span className="text-sm text-green-600">{t("common.copied")}</span>
          )}
        </div>
      </div>

      {/* Upload license */}
      {data.isSelfHostLicense && (
        <div className="w-full mt-4 flex flex-col">
          <label className="flex items-center gap-x-2">
            <span className="text-main">
              {t("subscription.upload-license")}
            </span>
          </label>
          <div className="mb-3 text-sm text-gray-400">
            {t("subscription.description")} {t("subscription.plan-compare")}{" "}
            <a
              href="https://www.bytebase.com/pricing?source=console"
              target="_blank"
              rel="noopener noreferrer"
              className="text-indigo-600 hover:underline"
            >
              {t("common.learn-more")} &gt;
            </a>
            {data.showTrial && allowEdit && (
              <button
                className="ml-1 text-indigo-600 hover:underline text-sm"
                onClick={onRequireEnterprise}
              >
                {t("subscription.plan.try")}
              </button>
            )}
          </div>
          <textarea
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm min-h-[100px] focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
            value={license}
            onChange={(e) => setLicense(e.target.value)}
            disabled={!allowEdit}
            placeholder={t("common.sensitive-placeholder")}
          />
          <div className="ml-auto mt-3">
            <button
              className="rounded-md bg-indigo-600 px-4 py-2 text-white capitalize hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
              disabled={disabled || !allowEdit}
              onClick={handleUpload}
            >
              {t("subscription.upload-license")}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function InstanceLicenseStats({
  planType,
  instanceCountLimit,
  activatedCount,
  totalLicenseCount,
  onManageInstanceLicenses,
}: {
  planType: string;
  instanceCountLimit: number;
  activatedCount: number;
  totalLicenseCount: string;
  onManageInstanceLicenses: () => void;
}) {
  const { t } = useTranslation();

  if (planType === "FREE") {
    return (
      <div className="flex flex-col text-left">
        <dt className="text-main">{t("subscription.max-instance-count")}</dt>
        <div className="mt-1 text-4xl">{instanceCountLimit}</div>
      </div>
    );
  }

  return (
    <>
      <div className="flex flex-col text-left">
        <dt className="text-main">
          {t("subscription.instance-assignment.used-and-total-license")}
        </dt>
        <div className="mt-1 text-4xl flex items-center gap-2">
          <span>
            {activatedCount}
            <span className="font-mono text-gray-500"> / </span>
            {totalLicenseCount}
          </span>
          <button
            className="text-gray-500 hover:text-gray-700"
            onClick={onManageInstanceLicenses}
          >
            <Pencil className="h-8 w-8" />
          </button>
        </div>
      </div>
    </>
  );
}
