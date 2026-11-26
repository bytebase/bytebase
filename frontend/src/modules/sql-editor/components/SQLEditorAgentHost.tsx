import { type ReactNode, useId, useLayoutEffect, useRef } from "react";
import {
  getRegisteredAgentSqlEditorHost,
  registerAgentSqlEditorHost,
} from "@/modules/agent/logic/sqlEditorHost";
import { useAgentStore } from "@/modules/agent/store/agent";
import { useAppStore } from "@/stores/app";
import { UNKNOWN_USER_NAME } from "@/types/v1/user";
import { captureSqlEditorContext } from "../model/sqlEditorAgentContext";
import { subscribeSQLEditorEditorState } from "../store/editor";
import { useSQLEditorTabsStore } from "../store/tab";

function getMatchingOwnerKey() {
  const { currentUser, currentUserName, workspace } = useAppStore.getState();
  const { workspaceName, userEmail } = useAgentStore.getState();
  if (
    !workspace?.name ||
    !currentUser?.email ||
    !currentUser.name ||
    currentUser.name === UNKNOWN_USER_NAME ||
    (currentUserName !== undefined && currentUserName !== currentUser.name) ||
    currentUser.workspace !== workspace.name ||
    workspaceName !== workspace.name ||
    userEmail !== currentUser.email
  ) {
    return;
  }
  return JSON.stringify([workspace.name, currentUser.name, userEmail]);
}

export function SQLEditorAgentHost({
  children,
  paneRatio,
}: {
  children: ReactNode;
  paneRatio: number;
}) {
  const id = useId();
  const slotRef = useRef<HTMLDivElement>(null);
  const mainColumnRef = useRef<HTMLDivElement>(null);
  const resizerRef = useRef<HTMLDivElement>(null);
  const unregisterRef = useRef<(() => void) | undefined>(undefined);

  useLayoutEffect(() => () => unregisterRef.current?.(), []);

  useLayoutEffect(() => {
    const slot = slotRef.current;
    const mainColumn = mainColumnRef.current;
    const resizer = resizerRef.current;
    if (!slot || !mainColumn || !resizer) return;

    let previousKey: string | undefined;
    const belongsToMount = (
      host: ReturnType<typeof getRegisteredAgentSqlEditorHost>
    ) => host?.id === id && host.slot === slot;
    const publish = () => {
      const owner = getMatchingOwnerKey();
      const registered = getRegisteredAgentSqlEditorHost();
      const ownsRegistration = belongsToMount(registered);
      if (!owner || !slot.isConnected) {
        if (ownsRegistration) unregisterRef.current?.();
        return;
      }
      if (registered && !ownsRegistration) return;

      const key = JSON.stringify([owner, captureSqlEditorContext()]);
      if (ownsRegistration && key === previousKey) return;
      previousKey = key;
      // Preserve the mounted host identity and replace its registration without
      // briefly exposing a missing host to the Agent.
      unregisterRef.current = registerAgentSqlEditorHost({
        id,
        slot,
        mainColumn,
        resizer,
        paneRatio,
        captureContext: () =>
          owner === getMatchingOwnerKey() &&
          belongsToMount(getRegisteredAgentSqlEditorHost())
            ? captureSqlEditorContext()
            : { kind: "pending" },
      });
    };
    const unsubscribeAgent = useAgentStore.subscribe((next, previous) => {
      if (
        next.workspaceName !== previous.workspaceName ||
        next.userEmail !== previous.userEmail
      ) {
        publish();
      }
    });
    const unsubscribeApp = useAppStore.subscribe(publish);
    const unsubscribeEditor = subscribeSQLEditorEditorState(publish);
    const unsubscribeTabs = useSQLEditorTabsStore.subscribe(
      (next, previous) => {
        const tab = next.tabsById.get(next.currentTabId);
        const previousTab = previous.tabsById.get(previous.currentTabId);
        if (
          tab?.id !== previousTab?.id ||
          tab?.mode !== previousTab?.mode ||
          tab?.connection !== previousTab?.connection ||
          tab?.viewState !== previousTab?.viewState ||
          tab?.batchQueryContext !== previousTab?.batchQueryContext
        ) {
          publish();
        }
      }
    );
    publish();
    return () => {
      unsubscribeAgent();
      unsubscribeApp();
      unsubscribeEditor();
      unsubscribeTabs();
    };
  }, [id, paneRatio]);

  return (
    <div ref={mainColumnRef} className="flex h-full min-w-0 overflow-hidden">
      <div className="flex min-w-0 flex-1 flex-col">{children}</div>
      <div ref={resizerRef} className="shrink-0" aria-hidden="true" />
      <div ref={slotRef} className="overflow-hidden" />
    </div>
  );
}
