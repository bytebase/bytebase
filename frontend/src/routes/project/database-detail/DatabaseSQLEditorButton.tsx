import { useTranslation } from "react-i18next";
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
  const { t } = useTranslation();
  const route = useCurrentRoute();
  const project = useProjectByName(database.project);

  return (
    <PermissionGuard permissions={["bb.sql.select"]} project={project}>
      {({ disabled: permissionDisabled }) => (
        <SQLEditorButton
          database={database}
          disabled={disabled || permissionDisabled}
          openInNewTab={!route.name?.startsWith("sql-editor")}
          appearance="solid"
          size="md"
          label={t("database.open-sql-editor")}
        />
      )}
    </PermissionGuard>
  );
}
