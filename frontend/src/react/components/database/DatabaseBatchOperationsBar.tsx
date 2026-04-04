import {
  ArrowRightLeft,
  Download,
  Pencil,
  RefreshCw,
  SquareStack,
  Tag,
  Unlink,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export interface DatabaseBatchOperationsBarProps {
  databases: Database[];
  onSyncSchema: () => void;
  onEditLabels: () => void;
  onEditEnvironment: () => void;
  // Global context: transfer to another project
  onTransferProject?: () => void;
  // Project context: unassign from current project
  onUnassign?: () => void;
  // Project context: change database schema
  onChangeDatabase?: () => void;
  // Project context: export data
  onExportData?: () => void;
}

export function DatabaseBatchOperationsBar({
  databases,
  onSyncSchema,
  onEditLabels,
  onEditEnvironment,
  onTransferProject,
  onUnassign,
  onChangeDatabase,
  onExportData,
}: DatabaseBatchOperationsBarProps) {
  const { t } = useTranslation();
  const canSync = hasWorkspacePermissionV2("bb.databases.sync");
  const canUpdate = hasWorkspacePermissionV2("bb.databases.update");
  const canGetEnvironment = hasWorkspacePermissionV2(
    "bb.settings.getEnvironment"
  );
  if (databases.length === 0) return null;
  return (
    <div className="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4">
      <span className="whitespace-nowrap">
        {t("database.selected-n-databases", { n: databases.length })}
      </span>
      <div className="flex items-center gap-x-2 flex-wrap">
        {onChangeDatabase && (
          <Button variant="ghost" size="sm" onClick={onChangeDatabase}>
            <Pencil className="h-4 w-4 mr-1" />
            {t("database.change-database")}
          </Button>
        )}
        {onExportData && (
          <Button variant="ghost" size="sm" onClick={onExportData}>
            <Download className="h-4 w-4 mr-1" />
            {t("custom-approval.risk-rule.risk.namespace.data_export")}
          </Button>
        )}
        <Button
          variant="ghost"
          size="sm"
          disabled={!canSync}
          onClick={onSyncSchema}
        >
          <RefreshCw className="h-4 w-4 mr-1" />
          {t("database.sync-schema-button")}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          disabled={!canUpdate}
          onClick={onEditLabels}
        >
          <Tag className="h-4 w-4 mr-1" />
          {t("database.edit-labels")}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          disabled={!canUpdate || !canGetEnvironment}
          onClick={onEditEnvironment}
        >
          <SquareStack className="h-4 w-4 mr-1" />
          {t("database.edit-environment")}
        </Button>
        {onTransferProject && (
          <Button
            variant="ghost"
            size="sm"
            disabled={!canUpdate}
            onClick={onTransferProject}
          >
            <ArrowRightLeft className="h-4 w-4 mr-1" />
            {t("database.transfer-project")}
          </Button>
        )}
        {onUnassign && (
          <Button
            variant="ghost"
            size="sm"
            disabled={!canUpdate}
            onClick={onUnassign}
          >
            <Unlink className="h-4 w-4 mr-1" />
            {t("database.unassign")}
          </Button>
        )}
      </div>
    </div>
  );
}
