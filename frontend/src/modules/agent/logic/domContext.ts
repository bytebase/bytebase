import type { AppRouterInstance } from "@/app/router";
import { getAppRouterState } from "@/app/router/navigation";
import { useAppStore } from "@/stores/app";
import type { AgentDomContext } from "../dom/interactionRegistry";
import { useAgentStore } from "../store/agent";
import {
  captureAgentExecutionContext,
  isAgentExecutionContextCurrent,
} from "./executionContext";

type Stage = Pick<
  AgentDomContext,
  "chatId" | "runId" | "runMode" | "projectName" | "signal" | "isCurrent"
>;
const observers = new WeakMap<AppRouterInstance, { generation: number }>();

function observe(router: AppRouterInstance) {
  const observer = { generation: 0 };
  const page = () => {
    const location = getAppRouterState()?.location;
    return JSON.stringify([
      router.currentRoute.value.fullPath,
      location && "key" in location ? location.key : undefined,
    ]);
  };
  const authority = () => {
    const principal = captureAgentExecutionContext();
    const agent = useAgentStore.getState();
    return JSON.stringify([
      principal?.actor,
      principal?.workspace,
      useAppStore.getState().currentUser?.email,
      agent.workspaceName,
      agent.userEmail,
    ]);
  };
  let lastPage = page(),
    lastAuthority = authority();
  router.afterEach(() => {
    const next = page();
    if (next !== lastPage) {
      lastPage = next;
      observer.generation++;
    }
  });
  const scopeChanged = () => {
    const next = authority();
    if (next !== lastAuthority) {
      lastAuthority = next;
      observer.generation++;
    }
  };
  useAppStore.subscribe(scopeChanged);
  useAgentStore.subscribe(scopeChanged);
  observers.set(router, observer);
  return observer;
}

/** Only an active page tool starts observing; a restored permission cannot recreate its authority. */
export function captureAgentDomContext(
  router: AppRouterInstance,
  stage: Stage,
  startObservation = false
): AgentDomContext | undefined {
  const principal = captureAgentExecutionContext();
  if (!principal || !stage.chatId || !stage.runId) return;
  const matchesAccount = () => {
    const agent = useAgentStore.getState();
    return (
      isAgentExecutionContextCurrent(principal) &&
      agent.workspaceName === principal.workspace &&
      !!agent.userEmail &&
      agent.userEmail === useAppStore.getState().currentUser?.email
    );
  };
  if (!matchesAccount()) return;
  const observer =
    observers.get(router) ?? (startObservation ? observe(router) : undefined);
  if (!observer) return;
  const pageKey = router.currentRoute.value.fullPath,
    pageGeneration = observer.generation;
  const isCurrent = () =>
    !stage.signal?.aborted &&
    stage.isCurrent() &&
    matchesAccount() &&
    router.currentRoute.value.fullPath === pageKey &&
    observer.generation === pageGeneration;
  return isCurrent()
    ? { ...stage, ...principal, pageKey, pageGeneration, isCurrent }
    : undefined;
}
