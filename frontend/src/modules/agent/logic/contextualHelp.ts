import { useAppStore } from "@/stores/app";
import { useAgentStore } from "../store/agent";
import type { AgentPageResource } from "./chatContext";

interface AgentPageHelpOptions {
  page: AgentPageResource;
  maskingRuleEditor?: boolean;
  onStart?: (prompt: string) => boolean;
  isAvailable?: () => boolean;
  canStart?: () => boolean;
}

export interface AgentPageHelpRequest extends AgentPageHelpOptions {
  workspaceName: string;
  userName: string;
}

type Listener = (request: AgentPageHelpRequest) => void;

const listeners = new Set<Listener>();
let currentRequest: AgentPageHelpRequest | undefined;
let pendingRequest: AgentPageHelpRequest | undefined;

function currentOwner() {
  const app = useAppStore.getState();
  const user = app.currentUser;
  const workspaceName =
    app.workspace?.name || user?.workspace || app.serverInfo?.workspace;
  if (
    !user?.name ||
    !workspaceName ||
    (user.workspace && user.workspace !== workspaceName)
  ) {
    return undefined;
  }
  return { workspaceName, userName: user.name };
}

function checkAvailability(check?: () => boolean): boolean {
  try {
    return check?.() !== false;
  } catch {
    return false;
  }
}

export function isAgentPageHelpRequestCurrent(
  request: AgentPageHelpRequest
): boolean {
  const owner = currentOwner();
  return (
    currentRequest === request &&
    owner?.workspaceName === request.workspaceName &&
    owner.userName === request.userName &&
    checkAvailability(request.isAvailable)
  );
}

export function canStartAgentPageHelp(request: AgentPageHelpRequest): boolean {
  return (
    isAgentPageHelpRequestCurrent(request) &&
    checkAvailability(request.canStart)
  );
}

export function subscribeAgentPageHelp(listener: Listener): () => void {
  listeners.add(listener);
  const pending = pendingRequest;
  pendingRequest = undefined;
  if (pending && isAgentPageHelpRequestCurrent(pending)) listener(pending);
  return () => listeners.delete(listener);
}

export function openAgentPageHelp(options: AgentPageHelpOptions): () => void {
  const owner = currentOwner();
  if (!owner || !checkAvailability(options.isAvailable)) return () => {};
  const request: AgentPageHelpRequest = {
    ...options,
    ...owner,
    page: structuredClone(options.page),
  };
  currentRequest = request;
  pendingRequest = listeners.size ? undefined : request;
  useAgentStore.getState().open();
  for (const listener of listeners) listener(request);
  return () => {
    if (currentRequest !== request) return;
    currentRequest = undefined;
    if (pendingRequest === request) pendingRequest = undefined;
    for (const listener of listeners) listener(request);
  };
}
