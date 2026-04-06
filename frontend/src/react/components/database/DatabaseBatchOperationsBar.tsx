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
import type { Permission } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE,
  PERMISSIONS_FOR_DATABASE_EXPORT_ISSUE,
} from "@/utils";

export interface DatabaseBatchOperationsBarProps {
  databases: Database[];
  /** When provided, permission checks use project-level IAM. */
  project?: Project;
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
  project,
  onSyncSchema,
  onEditLabels,
  onEditEnvironment,
  onTransferProject,
  onUnassign,
  onChangeDatabase,
  onExportData,
}: DatabaseBatchOperationsBarProps) {
  const { t } = useTranslation();

  const hasPermission = (permission: Permission) =>
    project
      ? hasProjectPermissionV2(project, permission)
      : hasWorkspacePermissionV2(permission);

  const canSync = hasPermission("bb.databases.sync");
  const canUpdate = hasPermission("bb.databases.update");
  const canGetEnvironment = hasPermission("bb.settings.getEnvironment");
  const canChangeDatabase =
    PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE.every(hasPermission);
  const canExportData =
    PERMISSIONS_FOR_DATABASE_EXPORT_ISSUE.every(hasPermission);

  if (databases.length === 0) return null;
  return (
    <div className="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4">
      <span className="whitespace-nowrap">
        {t("database.selected-n-databases", { n: databases.length })}
      </span>
      <div className="flex items-center gap-x-2 flex-wrap">
        {onChangeDatabase && (
          <Button
            variant="ghost"
            size="sm"
            disabled={!canChangeDatabase}
            onClick={onChangeDatabase}
          >
            <Pencil className="h-4 w-4 mr-1" />
            {t("database.change-database")}
          </Button>
        )}
        {onExportData && (
          <Button
            variant="ghost"
            size="sm"
            disabled={!canExportData}
            onClick={onExportData}
          >
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
