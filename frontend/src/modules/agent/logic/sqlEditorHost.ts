import { useAgentStore } from "../store/agent";
import {
  type AgentSqlContextCapture,
  normalizeAgentSqlEditorSnapshot,
} from "./sqlEditorContext";

export interface AgentSqlEditorHost {
  id: string;
  slot: HTMLElement;
  mainColumn: HTMLElement;
  resizer: HTMLElement;
  paneRatio: number;
  captureContext(): AgentSqlContextCapture;
}

export function registerAgentSqlEditorHost(host: AgentSqlEditorHost) {
  const snapshot = { ...host };
  const registration = () => snapshot;
  useAgentStore.setState({ sqlEditorHost: registration });
  return () => {
    if (useAgentStore.getState().sqlEditorHost === registration) {
      useAgentStore.setState({ sqlEditorHost: undefined });
    }
  };
}

export function getRegisteredAgentSqlEditorHost() {
  const host = useAgentStore.getState().sqlEditorHost?.();
  return host?.slot.isConnected ? host : undefined;
}

export function getCurrentAgentSqlEditorContext():
  | AgentSqlContextCapture
  | undefined {
  const host = getRegisteredAgentSqlEditorHost();
  if (!host) return undefined;
  const capture = host.captureContext();
  const snapshot =
    capture?.kind === "ready" &&
    normalizeAgentSqlEditorSnapshot(capture.snapshot);
  return snapshot ? { kind: "ready", snapshot } : { kind: "pending" };
}
