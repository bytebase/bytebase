import { LayersIcon, LinkIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/app/router/handles";
import { BytebaseLogo } from "@/components/BytebaseLogo";
import { usePermissionCheck } from "@/components/PermissionGuard";
import { useAppProject } from "@/hooks/useAppProject";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { isDarkTheme } from "./theme/derive";
import { useSQLEditorTheme } from "./theme/SQLEditorThemeScope";
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
  const theme = useSQLEditorTheme();
  const projectName = useSQLEditorEditorState((s) => s.project);

  const resolvedProject = useAppProject(projectName);
  const project = projectName ? resolvedProject : undefined;

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
      <BytebaseLogo builtinTheme={isDarkTheme(theme) ? "dark" : "light"} />
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
