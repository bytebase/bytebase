import { useRef } from "react";
import { AccessGrantRequestDrawer as SharedAccessGrantRequestDrawer } from "@/modules/access-control/AccessGrantRequestDrawer";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { getSQLEditorTabsState } from "@/modules/sql-editor/store/tab";
import { AccessGrant_Status } from "@/types/proto-es/v1/access_grant_service_pb";
import { useActiveSQLEditorTheme } from "./theme/useActiveSQLEditorTheme";

interface Props {
  readonly targets?: string[];
  readonly query?: string;
  readonly unmask?: boolean;
  readonly export?: boolean;
  readonly schema?: string;
  readonly container?: string;
  readonly reason?: string;
  readonly onClose: () => void;
}

export function AccessGrantRequestDrawer(props: Props) {
  const project = useSQLEditorEditorState((s) => s.project);
  const theme = useActiveSQLEditorTheme();
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const setHighlightAccessGrantName = useSQLEditorStore(
    (s) => s.setHighlightAccessGrantName
  );
  const initial = useRef<{
    targets: string[];
    schema?: string;
    container?: string;
  } | null>(null);
  if (!initial.current) {
    const tabs = getSQLEditorTabsState();
    const connection = tabs.tabsById.get(tabs.currentTabId)?.connection;
    initial.current = {
      targets: props.targets?.length
        ? props.targets
        : connection?.database
          ? [connection.database]
          : [],
      schema: props.schema ?? connection?.schema,
      container: props.container ?? connection?.table,
    };
  }
  return (
    <SharedAccessGrantRequestDrawer
      {...props}
      {...initial.current}
      project={project as string}
      theme={theme}
      onCreated={(grant) => {
        if (grant.status === AccessGrant_Status.PENDING && grant.issue) return;
        setAsidePanelTab("ACCESS");
        setHighlightAccessGrantName(grant.name);
      }}
    />
  );
}
