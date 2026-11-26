import {
  type ReactNode,
  useCallback,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { v4 as uuidv4 } from "uuid";
import {
  type AgentSqlEditorHost,
  getRegisteredAgentSqlEditorHost,
  registerAgentSqlEditorHost,
} from "@/modules/agent/logic/sqlEditorHost";
import { useAgentStore } from "@/modules/agent/store/agent";
import { useAppStore } from "@/stores/app";
import { UNKNOWN_USER_NAME } from "@/types/v1/user";
import { captureAgentQueryResultTarget } from "../model/agentQueryResult";
import {
  captureSqlEditorContext,
  getSqlEditorTargetScope,
} from "../model/sqlEditorAgentContext";
import {
  getSQLEditorEditorState,
  subscribeSQLEditorEditorState,
} from "../store/editor";
import { useSQLEditorTabsStore } from "../store/tab";
import {
  createSqlEditorAgentTarget,
  type SqlEditorTargetRegistration,
  SqlEditorTargetRegistrationContext,
} from "./StandardPanel/sqlEditorAgentTarget";

function getAppOwner() {
  const { currentUser, currentUserName, workspace } = useAppStore.getState();
  if (
    !workspace?.name ||
    !currentUser?.email ||
    !currentUser.name ||
    currentUser.name === UNKNOWN_USER_NAME ||
    (currentUserName !== undefined && currentUserName !== currentUser.name) ||
    currentUser.workspace !== workspace.name
  ) {
    return;
  }
  return {
    actor: currentUser.name,
    workspace: workspace.name,
    email: currentUser.email,
    key: JSON.stringify([workspace.name, currentUser.name, currentUser.email]),
  };
}

function getMatchingOwner() {
  const owner = getAppOwner();
  const { workspaceName, userEmail } = useAgentStore.getState();
  if (owner?.workspace === workspaceName && owner?.email === userEmail) {
    return owner;
  }
}

type MountedTarget = {
  tabId: string;
  ownerKey: string;
  project: string;
  adapter: ReturnType<typeof createSqlEditorAgentTarget>;
  unregister(): void;
};

export function SQLEditorAgentHost({
  children,
  paneRatio,
}: {
  children: ReactNode;
  paneRatio: number;
}) {
  const [id] = useState(() => uuidv4());
  const slotRef = useRef<HTMLDivElement>(null);
  const mainColumnRef = useRef<HTMLDivElement>(null);
  const resizerRef = useRef<HTMLDivElement>(null);
  const unregisterRef = useRef<(() => void) | undefined>(undefined);
  const nativeRef = useRef<MountedTarget | undefined>(undefined);
  const publishNativeRef = useRef<(() => void) | undefined>(undefined);

  const registerTarget = useCallback<SqlEditorTargetRegistration>(
    (tabId, editor, readOnlyOption) => {
      const owner = getAppOwner();
      if (!owner) return () => {};
      nativeRef.current?.unregister();
      const publish = () => publishNativeRef.current?.();
      const native: MountedTarget = {
        tabId,
        ownerKey: owner.key,
        project: getSQLEditorEditorState().project,
        adapter: createSqlEditorAgentTarget(editor, readOnlyOption),
        unregister() {
          if (nativeRef.current !== native) return;
          nativeRef.current = undefined;
          for (const subscription of subscriptions) subscription.dispose();
          publish();
        },
      };
      const subscriptions = [
        editor.onDidDispose(native.unregister),
        editor.onDidChangeModelContent(publish),
        editor.onDidChangeCursorSelection(publish),
        editor.onDidChangeModel(publish),
        editor.onDidChangeConfiguration((event) => {
          if (event.hasChanged(readOnlyOption)) publish();
        }),
      ];
      nativeRef.current = native;
      publish();
      return native.unregister;
    },
    []
  );

  useLayoutEffect(
    () => () => {
      publishNativeRef.current = undefined;
      nativeRef.current?.unregister();
      unregisterRef.current?.();
    },
    []
  );

  useLayoutEffect(() => {
    const slot = slotRef.current;
    const mainColumn = mainColumnRef.current;
    const resizer = resizerRef.current;
    if (!slot || !mainColumn || !resizer) return;

    let previousKey: string | undefined;
    const belongsToMount = (
      host: ReturnType<typeof getRegisteredAgentSqlEditorHost>
    ) => host?.id === id && host.slot === slot;
    const readNative = (
      owner: NonNullable<ReturnType<typeof getMatchingOwner>>
    ) => {
      const native = nativeRef.current;
      if (!native) return { status: "unavailable" } as const;
      const live = getSqlEditorTargetScope({
        actor: owner.actor,
        workspace: owner.workspace,
        hostId: id,
      });
      if (live.status !== "ready") return live;
      if (
        native.ownerKey !== owner.key ||
        native.tabId !== live.scope.tabId ||
        native.project !== (live.scope.project ?? "")
      ) {
        return { status: "stale" } as const;
      }
      return { ...live, adapter: native.adapter };
    };
    const capture = (
      owner: NonNullable<ReturnType<typeof getMatchingOwner>>
    ) => {
      const native = readNative(owner);
      return captureSqlEditorContext(
        native.status === "ready"
          ? native.adapter.capture(native.scope, native.statement)
          : undefined
      );
    };
    const publish = () => {
      const owner = getMatchingOwner();
      const registered = getRegisteredAgentSqlEditorHost();
      const ownsRegistration = belongsToMount(registered);
      if (!owner || !slot.isConnected) {
        if (ownsRegistration) unregisterRef.current?.();
        return;
      }
      if (registered && !ownsRegistration) return;

      const key = JSON.stringify([owner.key, capture(owner)]);
      if (ownsRegistration && key === previousKey) return;
      previousKey = key;
      const isCurrent = () =>
        owner.key === getMatchingOwner()?.key &&
        belongsToMount(getRegisteredAgentSqlEditorHost());
      const inspect: NonNullable<AgentSqlEditorHost["inspect"]> = (target) => {
        if (!isCurrent()) return { status: "unavailable" };
        const live = readNative(owner);
        return live.status === "ready"
          ? live.adapter.inspect(live.scope, target, live.statement)
          : { status: live.status };
      };
      const applySql: NonNullable<AgentSqlEditorHost["applySql"]> = (
        request
      ) => {
        if (!isCurrent()) return { status: "unavailable" };
        const live = readNative(owner);
        return live.status === "ready"
          ? live.adapter.applySql(live.scope, request, live.statement)
          : { status: live.status };
      };
      // Preserve the mounted host identity and replace its registration without
      // briefly exposing a missing host to the Agent.
      unregisterRef.current = registerAgentSqlEditorHost({
        id,
        slot,
        mainColumn,
        resizer,
        paneRatio,
        inspect,
        applySql,
        captureQueryResultTarget: captureAgentQueryResultTarget,
        captureContext: () =>
          isCurrent() ? capture(owner) : { kind: "pending" },
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
          tab?.savedQuery !== previousTab?.savedQuery ||
          tab?.statement !== previousTab?.statement ||
          tab?.connection !== previousTab?.connection ||
          tab?.viewState !== previousTab?.viewState ||
          tab?.batchQueryContext !== previousTab?.batchQueryContext
        ) {
          publish();
        }
      }
    );
    publishNativeRef.current = publish;
    publish();
    return () => {
      publishNativeRef.current = undefined;
      unsubscribeAgent();
      unsubscribeApp();
      unsubscribeEditor();
      unsubscribeTabs();
    };
  }, [id, paneRatio]);

  return (
    <div ref={mainColumnRef} className="flex h-full min-w-0 overflow-hidden">
      <div className="flex min-w-0 flex-1 flex-col">
        <SqlEditorTargetRegistrationContext.Provider value={registerTarget}>
          {children}
        </SqlEditorTargetRegistrationContext.Provider>
      </div>
      <div ref={resizerRef} className="shrink-0" aria-hidden="true" />
      <div ref={slotRef} className="overflow-hidden" />
    </div>
  );
}
