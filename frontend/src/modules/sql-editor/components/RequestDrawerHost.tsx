import {
  createContext,
  type ReactNode,
  useContext,
  useMemo,
  useRef,
  useState,
} from "react";
import { RequestRoleSheet } from "@/modules/access-control/RequestRoleSheet";
import type { DatabaseResource, Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { AccessGrantRequestDrawer } from "./AccessGrantRequestDrawer";
import { useActiveSQLEditorTheme } from "./theme/useActiveSQLEditorTheme";

type RequestRoleArgs = {
  project: Project;
  requiredPermissions: Permission[];
  initialRole: string;
  initialDatabaseResources: DatabaseResource[];
};

type AccessGrantArgs = {
  query?: string;
  targets: string[];
  schema?: string;
  container?: string;
  unmask?: boolean;
  export?: boolean;
};

type ContextValue = {
  openRequestRoleSheet: (args: RequestRoleArgs) => boolean;
  openAccessGrantDrawer: (args: AccessGrantArgs) => boolean;
};

const RequestDrawerHostContext = createContext<ContextValue | null>(null);

export function useRequestDrawerHost(): ContextValue | null {
  return useContext(RequestDrawerHostContext);
}

/**
 * Hosts the role-request and JIT-grant drawers at the SQL Editor layout
 * level so they survive when descendant overlays (e.g. the connection
 * panel Sheet) close, and so their own Sheet backdrop renders above the
 * connection panel — giving the user the expected modal cover.
 *
 * `RequestQueryButton` calls `useRequestDrawerHost().openX(...)` instead
 * of holding local drawer state, which previously caused the drawer to
 * unmount whenever its host row/connection panel did.
 */
export function RequestDrawerHost({ children }: { children: ReactNode }) {
  const [roleArgs, setRoleArgs] = useState<RequestRoleArgs | null>(null);
  const [grantArgs, setGrantArgs] = useState<AccessGrantArgs | null>(null);
  const requestOpenRef = useRef(false);

  const value = useMemo<ContextValue>(
    () => ({
      openRequestRoleSheet: (args) => {
        if (requestOpenRef.current) return false;
        requestOpenRef.current = true;
        setRoleArgs(args);
        return true;
      },
      openAccessGrantDrawer: (args) => {
        if (requestOpenRef.current) return false;
        requestOpenRef.current = true;
        setGrantArgs(args);
        return true;
      },
    }),
    []
  );

  // This host only renders inside the SQL Editor, so resolve the ACTIVE theme
  // (the dark admin fallback in admin mode) and hand it to RequestRoleSheet as
  // an explicit prop. RequestRoleSheet is shared with non-SQL-Editor pages, so
  // it stays light unless a host passes a theme — this is the only caller that
  // does. (AccessGrantRequestDrawer is SQL-Editor-only and reads the active
  // theme itself.)
  const activeTheme = useActiveSQLEditorTheme();

  return (
    <RequestDrawerHostContext.Provider value={value}>
      {children}
      {roleArgs && (
        <RequestRoleSheet
          open
          project={roleArgs.project}
          requiredPermissions={roleArgs.requiredPermissions}
          initialRole={roleArgs.initialRole}
          initialDatabaseResources={roleArgs.initialDatabaseResources}
          theme={activeTheme}
          onClose={() => {
            requestOpenRef.current = false;
            setRoleArgs(null);
          }}
        />
      )}
      {grantArgs && (
        <AccessGrantRequestDrawer
          query={grantArgs.query}
          targets={grantArgs.targets}
          schema={grantArgs.schema}
          container={grantArgs.container}
          unmask={grantArgs.unmask}
          export={grantArgs.export}
          onClose={() => {
            requestOpenRef.current = false;
            setGrantArgs(null);
          }}
        />
      )}
    </RequestDrawerHostContext.Provider>
  );
}
