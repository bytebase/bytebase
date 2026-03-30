import { Copy, Pencil } from "lucide-react";
import { type ChangeEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { Textarea } from "@/react/components/ui/textarea";
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
              <Badge variant="destructive" className="ml-2 h-6">
                {t("subscription.expired")}
              </Badge>
            )}
            {!data.isExpired && data.isTrialing && (
              <Badge variant="secondary" className="ml-2 h-6">
                {t("subscription.trialing")}
              </Badge>
            )}
          </div>
          <div className="text-accent mt-1 text-3xl lg:text-4xl">
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
              <Button className="text-lg" onClick={onRequireEnterprise}>
                {t("subscription.enterprise-free-trial", {
                  days: data.trialingDays,
                })}
              </Button>
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
              <Button className="text-lg" onClick={onRequireEnterprise}>
                {t("subscription.contact-us")}
              </Button>
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
            <span className="font-mono text-control-light">/</span>
            {userLimit}
          </div>
        </div>
      </div>

      {/* Divider */}
      <hr className="my-6 border-t border-block-border" />

      {/* Workspace ID */}
      <div>
        <label className="flex items-center gap-x-2">
          <span className="text-main">
            {t("settings.general.workspace.id")}
          </span>
        </label>
        <div className="mb-3 text-sm text-control-placeholder">
          {t("settings.general.workspace.id-description")}
        </div>
        <div className="mb-4 flex items-center gap-x-2">
          <Input
            readOnly
            value={data.workspaceId}
            onClick={(e) => (e.target as HTMLInputElement).select()}
          />
          <Button
            variant="ghost"
            size="icon"
            onClick={handleCopy}
            title={t("common.copy")}
          >
            <Copy className="h-4 w-4" />
          </Button>
          {copied && (
            <span className="text-sm text-success">{t("common.copied")}</span>
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
          <div className="mb-3 text-sm text-control-placeholder">
            {t("subscription.description")} {t("subscription.plan-compare")}{" "}
            <a
              href="https://www.bytebase.com/pricing?source=console"
              target="_blank"
              rel="noopener noreferrer"
              className="text-accent hover:underline"
            >
              {t("common.learn-more")} &gt;
            </a>
            {data.showTrial && allowEdit && (
              <Button
                variant="link"
                size="sm"
                className="ml-1"
                onClick={onRequireEnterprise}
              >
                {t("subscription.plan.try")}
              </Button>
            )}
          </div>
          <Textarea
            value={license}
            onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
              setLicense(e.target.value)
            }
            disabled={!allowEdit}
            placeholder={t("common.sensitive-placeholder")}
          />
          <div className="ml-auto mt-3">
            <Button
              disabled={disabled || !allowEdit}
              onClick={handleUpload}
              className="capitalize"
            >
              {t("subscription.upload-license")}
            </Button>
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
            <span className="font-mono text-control-light"> / </span>
            {totalLicenseCount}
          </span>
          <Button
            variant="ghost"
            size="icon"
            onClick={onManageInstanceLicenses}
          >
            <Pencil className="h-5 w-5" />
          </Button>
        </div>
      </div>
    </>
  );
}
