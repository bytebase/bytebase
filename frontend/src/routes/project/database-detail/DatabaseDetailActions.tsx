import { ArrowRightLeft, Download, Pencil, RefreshCw } from "lucide-react";
import { type ReactNode, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PermissionGuard } from "@/components/PermissionGuard";
import { useProjectByName } from "@/hooks/useProjectByName";
import { preCreateIssue } from "@/lib/plan/issue";
import { useAppStore } from "@/stores/app";
import type { Permission } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { isProjectInstanceDatabase } from "@/utils/v1/database";
import { type DatabaseAction, DatabaseActionBar } from "./DatabaseActionBar";
import { useDatabaseSchemaExport } from "./DatabaseExportSchemaButton";
import { DatabaseSQLEditorButton } from "./DatabaseSQLEditorButton";
import { useDatabaseSync } from "./DatabaseSyncButton";

const DATABASE_CHANGE_PERMISSIONS: Permission[] = [
  "bb.plans.create",
  "bb.sheets.create",
];

export function DatabaseDetailActions({
  database,
  isDefaultProject,
  onOpenTransferProject,
}: {
  database: Database;
  isDefaultProject: boolean;
  onOpenTransferProject: () => void;
}) {
  const { t } = useTranslation();
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  const hasProjectPermissionFn = useAppStore(
    (state) => state.hasProjectPermission
  );
  const hasWorkspacePermissionFn = useAppStore(
    (state) => state.hasWorkspacePermission
  );
  const project = useProjectByName(database.project);
  void projectsByName;
  const hasProjectPermission = useMemo(
    () => (permission: Permission) => {
      if (hasWorkspacePermissionFn(permission)) {
        return true;
      }
      return project ? hasProjectPermissionFn(project, permission) : false;
    },
    [hasProjectPermissionFn, hasWorkspacePermissionFn, project]
  );

  const canUpdate = useMemo(
    () => hasProjectPermission("bb.databases.update"),
    [hasProjectPermission]
  );
  const canChangeDatabase = useMemo(
    () =>
      DATABASE_CHANGE_PERMISSIONS.every((permission) =>
        hasProjectPermission(permission)
      ),
    [hasProjectPermission]
  );
  const canSync = useMemo(
    () => hasProjectPermission("bb.databases.sync"),
    [hasProjectPermission]
  );
  const canExportSchema = useMemo(
    () => hasProjectPermission("bb.databases.getSchema"),
    [hasProjectPermission]
  );

  const { syncing, sync } = useDatabaseSync(database);
  const { exporting, options, exportSchema } =
    useDatabaseSchemaExport(database);
  const guard = (permissions: Permission[]) => (content: ReactNode) => (
    <PermissionGuard permissions={permissions} project={project}>
      {content}
    </PermissionGuard>
  );
  const actions: DatabaseAction[] = [];
  if (!isDefaultProject) {
    actions.push({
      key: "change",
      label: t("database.change-database"),
      icon: Pencil,
      disabled: !canChangeDatabase,
      onClick: () => void preCreateIssue(database.project, [database.name]),
      wrap: guard(DATABASE_CHANGE_PERMISSIONS),
    });
  }
  actions.push(
    {
      key: "sync",
      label: t("database.sync-database"),
      icon: RefreshCw,
      disabled: !canSync || syncing,
      onClick: () => void sync(),
      wrap: guard(["bb.databases.sync"]),
    },
    {
      key: "export",
      label: t("database.export-schema"),
      icon: Download,
      disabled: !canExportSchema || exporting,
      options: options.map((option) => ({
        ...option,
        onClick: () => void exportSchema(option.key),
      })),
      wrap: guard(["bb.databases.getSchema"]),
    }
  );
  if (!isDefaultProject && !isProjectInstanceDatabase(database)) {
    actions.push({
      key: "transfer",
      label: t("database.transfer-project"),
      icon: ArrowRightLeft,
      disabled: !canUpdate,
      onClick: onOpenTransferProject,
      wrap: guard(["bb.databases.update"]),
    });
  }
  return (
    <DatabaseActionBar
      actions={actions}
      primary={<DatabaseSQLEditorButton database={database} />}
    />
  );
}
