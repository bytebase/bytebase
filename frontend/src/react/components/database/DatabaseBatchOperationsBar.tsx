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
import {
  type SelectionAction,
  SelectionActionBar,
} from "@/react/components/SelectionActionBar";
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
  /**
   * True when every visible database on the current page is selected.
   * Drives the leading checkbox's checked vs. indeterminate state.
   */
  allSelected: boolean;
  /**
   * Toggles between "select every visible database" and "clear selection".
   */
  onToggleSelectAll: () => void;
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
  allSelected,
  onToggleSelectAll,
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

  const actions: SelectionAction[] = [
    {
      key: "change-database",
      label: t("database.change-database"),
      icon: Pencil,
      onClick: () => onChangeDatabase?.(),
      disabled: !canChangeDatabase,
      hidden: !onChangeDatabase,
    },
    {
      key: "export-data",
      label: t("custom-approval.risk-rule.risk.namespace.data_export"),
      icon: Download,
      onClick: () => onExportData?.(),
      disabled: !canExportData,
      hidden: !onExportData,
    },
    {
      key: "sync-schema",
      label: t("database.sync-schema-button"),
      icon: RefreshCw,
      onClick: onSyncSchema,
      disabled: !canSync,
    },
    {
      key: "edit-labels",
      label: t("database.edit-labels"),
      icon: Tag,
      onClick: onEditLabels,
      disabled: !canUpdate,
    },
    {
      key: "edit-environment",
      label: t("database.edit-environment"),
      icon: SquareStack,
      onClick: onEditEnvironment,
      disabled: !canUpdate || !canGetEnvironment,
    },
    {
      key: "transfer-project",
      label: t("database.transfer-project"),
      icon: ArrowRightLeft,
      onClick: () => onTransferProject?.(),
      disabled: !canUpdate,
      hidden: !onTransferProject,
    },
    {
      key: "unassign",
      label: t("database.unassign"),
      icon: Unlink,
      onClick: () => onUnassign?.(),
      disabled: !canUpdate,
      hidden: !onUnassign,
    },
  ];

  return (
    <SelectionActionBar
      count={databases.length}
      label={t("common.n-selected", { n: databases.length })}
      allSelected={allSelected}
      onToggleSelectAll={onToggleSelectAll}
      actions={actions}
    />
  );
}
