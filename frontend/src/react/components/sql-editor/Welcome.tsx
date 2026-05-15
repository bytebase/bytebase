import { LayersIcon, LinkIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { BytebaseLogo } from "@/react/components/BytebaseLogo";
import { usePermissionCheck } from "@/react/components/PermissionGuard";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import { router } from "@/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { useProjectV1Store } from "@/store";
import { WelcomeButton } from "./WelcomeButton";

export type WelcomeProps = {
  /**
   * Called when the user clicks "Connect to a database". Vue parent
   * passes a callback that sets `asidePanelTab = "SCHEMA"` and
   * `showConnectionPanel = true` on the Vue-side SQL Editor context.
   */
  readonly onChangeConnection: () => void;
};

export function Welcome({ onChangeConnection }: WelcomeProps) {
  const { t } = useTranslation();
  const sqlEditorStore = useSQLEditorVueState();
  const projectV1Store = useProjectV1Store();

  const project = useVueState(() => {
    const projectName = sqlEditorStore.project;
    return projectName
      ? projectV1Store.getProjectByName(projectName)
      : undefined;
  });

  const [showCreateInstanceButton] = usePermissionCheck([
    "bb.instances.create",
  ]);
  const [showConnectButton] = usePermissionCheck(["bb.sql.select"], project);

  const handleCreateInstance = () => {
    router.push({
      name: INSTANCE_ROUTE_DASHBOARD,
      hash: "#add",
    });
  };

  return (
    <div className="w-full h-full flex flex-col items-center justify-center gap-y-4">
      <BytebaseLogo />
      <div className="flex items-center flex-wrap gap-4">
        {showCreateInstanceButton && (
          <WelcomeButton
            variant="secondary"
            icon={<LayersIcon strokeWidth={1.5} className="size-8" />}
            onClick={handleCreateInstance}
          >
            {t("sql-editor.add-a-new-instance")}
          </WelcomeButton>
        )}
        {showConnectButton && (
          <WelcomeButton
            variant="primary"
            icon={<LinkIcon strokeWidth={1.5} className="size-8" />}
            onClick={onChangeConnection}
          >
            {t("sql-editor.connect-to-a-database")}
          </WelcomeButton>
        )}
      </div>
    </div>
  );
}
