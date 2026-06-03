import { Download } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { useAppProject } from "@/react/hooks/useAppProject";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { AccessGrantRequestDrawer } from "./AccessGrantRequestDrawer";
import { useRequestDrawerHost } from "./RequestDrawerHost";

interface Props {
  readonly size?: "sm" | "default";
  readonly statement?: string;
  readonly targets: string[];
}

/**
 * Shown in the result panel when the query-data policy disables export but
 * the project allows just-in-time access. Clicking it opens the access-grant
 * drawer pre-filled with the current database, statement, and both the unmask
 * and export capabilities checked.
 */
export function RequestExportButton({
  size = "sm",
  statement,
  targets,
}: Props) {
  const { t } = useTranslation();

  // When the layout-level host is mounted (typical case inside the SQL
  // Editor), opening the drawer dispatches up to the host so it survives
  // ancestor unmounts. Local state stays as a fallback for standalone callers.
  const drawerHost = useRequestDrawerHost();
  const [showDrawer, setShowDrawer] = useState(false);

  const loadSubscription = useAppStore((s) => s.loadSubscription);
  // The SQL editor route doesn't mount the dashboard shells that load the
  // app-store subscription, so load it here before reading feature gates.
  useEffect(() => {
    void loadSubscription();
  }, [loadSubscription]);

  const projectName = useSQLEditorEditorState((s) => s.project);
  const project = useAppProject(projectName);
  const hasJITFeature = useAppStore((s) =>
    s.hasFeature(PlanFeature.FEATURE_JIT)
  );

  if (!project?.allowJustInTimeAccess) {
    return null;
  }

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (drawerHost) {
      drawerHost.openAccessGrantDrawer({
        query: statement,
        targets,
        unmask: true,
        export: true,
      });
    } else {
      setShowDrawer(true);
    }
  };

  return (
    <div>
      <PermissionGuard
        permissions={["bb.accessGrants.create"]}
        project={project}
      >
        {({ disabled }) => (
          <Button
            size={size}
            variant="default"
            disabled={disabled || !hasJITFeature}
            onClick={handleClick}
            className={cn("gap-x-1")}
          >
            {hasJITFeature ? (
              <Download className="size-4" />
            ) : (
              <FeatureBadge
                clickable={false}
                feature={PlanFeature.FEATURE_JIT}
              />
            )}
            {t("sql-editor.request-export")}
          </Button>
        )}
      </PermissionGuard>

      {showDrawer && (
        <AccessGrantRequestDrawer
          query={statement}
          targets={targets}
          unmask={true}
          export={true}
          onClose={() => setShowDrawer(false)}
        />
      )}
    </div>
  );
}
