import { ArrowRightLeft, Pencil } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { preCreateIssue } from "@/components/Plan/logic/issue";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { usePermissionStore, useProjectV1Store } from "@/store";
import type { Permission } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseExportSchemaButton } from "./DatabaseExportSchemaButton";
import { DatabaseSyncButton } from "./DatabaseSyncButton";

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
  const projectStore = useProjectV1Store();
  const permissionStore = usePermissionStore();
  const project = useVueState(() =>
    projectStore.getProjectByName(database.project)
  );
  const hasProjectPermission = useMemo(
    () => (permission: Permission) => {
      if (permissionStore.currentPermissions.has(permission)) {
        return true;
      }
      return project
        ? permissionStore.currentPermissionsInProjectV1(project).has(permission)
        : false;
    },
    [permissionStore, project]
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

  return (
    <>
      <div className="flex shrink-0 flex-wrap items-center justify-start gap-x-2 gap-y-2">
        <DatabaseSyncButton database={database} disabled={!canSync} />
        <DatabaseExportSchemaButton
          database={database}
          disabled={!canExportSchema}
        />
        {!isDefaultProject && (
          <Button
            variant="outline"
            disabled={!canUpdate}
            onClick={onOpenTransferProject}
          >
            <ArrowRightLeft className="h-4 w-4" />
            {t("database.transfer-project")}
          </Button>
        )}
        {!isDefaultProject && (
          <Button
            variant="outline"
            disabled={!canChangeDatabase}
            onClick={() =>
              void preCreateIssue(database.project, [database.name])
            }
          >
            <Pencil className="h-4 w-4" />
            {t("database.change-database")}
          </Button>
        )}
      </div>
    </>
  );
}
