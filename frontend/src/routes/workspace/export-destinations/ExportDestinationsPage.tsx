import { create } from "@bufbuild/protobuf";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import sharePointIcon from "@/assets/export/sharepoint.svg";
import { FeatureAttention } from "@/components/FeatureAttention";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  WorkspacePageInfo,
  WorkspacePageLayout,
} from "@/components/WorkspacePageLayout";
import { usePlanFeature } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
import {
  Setting_SettingName,
  type SharePointSetting,
  SharePointSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { SharePointSheet } from "./SharePointSheet";

function getStoredSharePointSetting(): SharePointSetting {
  const setting = useAppStore
    .getState()
    .getSettingByName(Setting_SettingName.EXPORT);
  const value = setting?.value?.value;
  if (value?.case !== "export" || !value.value.sharepoint) {
    return create(SharePointSettingSchema, {});
  }
  return value.value.sharepoint;
}

export function ExportDestinationsPage() {
  const { t } = useTranslation();
  const settingsByName = useAppStore((s) => s.settingsByName);
  const sharePoint = useMemo(
    () => getStoredSharePointSetting(),
    [settingsByName]
  );
  const hasFeature = usePlanFeature(PlanFeature.FEATURE_SHAREPOINT_EXPORT);
  const allowEdit = hasWorkspacePermissionV2("bb.settings.set") && hasFeature;
  const [initialized, setInitialized] = useState(false);
  const [sharePointOpen, setSharePointOpen] = useState(false);
  const [sessionKey, setSessionKey] = useState(0);

  useEffect(() => {
    void useAppStore
      .getState()
      .getOrFetchSettingByName(Setting_SettingName.EXPORT, true)
      .finally(() => setInitialized(true));
  }, []);

  const openSharePoint = () => {
    setSessionKey((key) => key + 1);
    setSharePointOpen(true);
  };

  return (
    <WorkspacePageLayout>
      <WorkspacePageInfo
        description={t("settings.export-destinations.description")}
      />
      <FeatureAttention feature={PlanFeature.FEATURE_SHAREPOINT_EXPORT} />
      {initialized && (
        <div className="overflow-hidden rounded-sm border border-block-border">
          <Table>
            <TableHeader>
              <TableRow className="bg-control-bg">
                <TableHead>
                  {t("settings.export-destinations.provider")}
                </TableHead>
                <TableHead>{t("common.status")}</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              <DestinationRow
                icon={sharePointIcon}
                title={t("export-data.destination.sharepoint")}
                caption={t("settings.sharepoint.caption")}
                configured={sharePoint.tenantId !== ""}
                allowEdit={allowEdit}
                onOpen={openSharePoint}
              />
            </TableBody>
          </Table>
        </div>
      )}
      <SharePointSheet
        open={sharePointOpen}
        sessionKey={sessionKey}
        stored={sharePoint}
        allowEdit={allowEdit}
        onClose={() => setSharePointOpen(false)}
      />
    </WorkspacePageLayout>
  );
}

function DestinationRow({
  icon,
  title,
  caption,
  configured,
  allowEdit,
  onOpen,
}: {
  icon: string;
  title: string;
  caption: string;
  configured: boolean;
  allowEdit: boolean;
  onOpen: () => void;
}) {
  const { t } = useTranslation();
  const action = !configured
    ? t("common.configure")
    : allowEdit
      ? t("common.edit")
      : t("common.view");
  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-x-3">
          <img src={icon} alt="" className="size-6" />
          <div className="flex flex-col">
            <span className="font-medium text-main">{title}</span>
            <span className="text-xs text-control-light">{caption}</span>
          </div>
        </div>
      </TableCell>
      <TableCell>
        {configured ? (
          <Badge variant="success">
            {t("settings.export-destinations.configured")}
          </Badge>
        ) : (
          <Badge>{t("settings.export-destinations.not-configured")}</Badge>
        )}
      </TableCell>
      <TableCell className="text-right">
        <Button
          appearance="outline"
          size="sm"
          aria-label={`${action} ${title}`}
          disabled={!configured && !allowEdit}
          onClick={onOpen}
        >
          {action}
        </Button>
      </TableCell>
    </TableRow>
  );
}
