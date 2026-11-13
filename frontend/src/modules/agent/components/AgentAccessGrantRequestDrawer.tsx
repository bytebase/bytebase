import type { ComponentProps } from "react";
import { useReactiveRoute } from "@/hooks/useReactiveRoute";
import { AccessGrantRequestDrawer } from "@/modules/access-control/AccessGrantRequestDrawer";
import { useActiveSQLEditorTheme } from "@/modules/sql-editor/components/theme/useActiveSQLEditorTheme";

export function AgentAccessGrantRequestDrawer(
  props: ComponentProps<typeof AccessGrantRequestDrawer>
) {
  const route = useReactiveRoute();
  const theme = useActiveSQLEditorTheme();
  return (
    <AccessGrantRequestDrawer
      {...props}
      theme={route.name?.startsWith("sql-editor.") ? theme : undefined}
    />
  );
}
