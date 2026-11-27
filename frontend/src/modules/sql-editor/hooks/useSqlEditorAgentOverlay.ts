import { useLayoutEffect } from "react";
import { getRegisteredAgentSqlEditorHost } from "@/modules/agent/logic/sqlEditorHost";
import { useAgentStore } from "@/modules/agent/store/agent";

interface OverlaySession {
  workspace: string;
  email: string;
  hostId: string;
  holders: Set<symbol>;
  restore: boolean;
  unsubscribe: () => void;
}

let active: OverlaySession | undefined;

function yieldToOverlay() {
  const store = useAgentStore.getState();
  const host = getRegisteredAgentSqlEditorHost();
  if (!host || !store.workspaceName || !store.userEmail) return;
  let session = active;
  if (
    !session ||
    session.hostId !== host.id ||
    session.workspace !== store.workspaceName ||
    session.email !== store.userEmail
  ) {
    if (!store.visible || store.minimized) return;
    const owner: OverlaySession = {
      workspace: store.workspaceName,
      email: store.userEmail,
      hostId: host.id,
      holders: new Set(),
      restore: true,
      unsubscribe: () => {},
    };
    active = session = owner;
    store.minimize();
    // An explicit restore/close or account change wins, even if the same
    // minimized state returns before the final native overlay closes.
    owner.unsubscribe = useAgentStore.subscribe((next) => {
      if (
        !next.visible ||
        !next.minimized ||
        next.workspaceName !== owner.workspace ||
        next.userEmail !== owner.email
      )
        owner.restore = false;
    });
  }
  const current = session;
  const token = Symbol();
  current.holders.add(token);
  return () => {
    if (!current.holders.delete(token) || current.holders.size) return;
    current.unsubscribe();
    if (active !== current) return;
    active = undefined;
    const next = useAgentStore.getState();
    if (
      current.restore &&
      next.visible &&
      next.minimized &&
      next.workspaceName === current.workspace &&
      next.userEmail === current.email
    )
      next.restore();
  };
}

export function useSqlEditorAgentOverlay(open: boolean) {
  useLayoutEffect(() => {
    if (open) return yieldToOverlay();
  }, [open]);
}
