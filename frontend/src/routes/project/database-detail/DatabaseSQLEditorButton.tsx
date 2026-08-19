import { useCurrentRoute } from "@/app/router";
import { PermissionGuard } from "@/components/PermissionGuard";
import { SQLEditorButton } from "@/components/SQLEditorButton";
import { useProjectByName } from "@/hooks/useProjectByName";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export function DatabaseSQLEditorButton({
  database,
  disabled = false,
}: {
  database: Database;
  disabled?: boolean;
}) {
  const route = useCurrentRoute();
  const project = useProjectByName(database.project);

  return (
    <dd className="md:mr-4">
      <PermissionGuard permissions={["bb.sql.select"]} project={project}>
        {({ disabled: permissionDisabled }) => (
          <SQLEditorButton
            database={database}
            disabled={disabled || permissionDisabled}
            openInNewTab={!route.name?.startsWith("sql-editor")}
            appearance="secondary"
            size="sm"
          />
        )}
      </PermissionGuard>
    </dd>
  );
}
