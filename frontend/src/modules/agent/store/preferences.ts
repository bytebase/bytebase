import { getResourceId, workspaceNamePrefix } from "@/lib/resourceName";

export interface AgentPreferences {
  sendWithEnter: boolean;
  tokenLimitPerRun: number;
  openingBehavior: "landing" | "continue";
}

export const DEFAULT_AGENT_PREFERENCES: Readonly<AgentPreferences> = {
  sendWithEnter: true,
  tokenLimitPerRun: 100_000,
  openingBehavior: "landing",
};

const AGENT_PREFERENCES_KEY = "bb-agent-preferences-v1";

export const getAgentPreferencesKey = (
  workspaceName: string,
  userEmail: string
) => {
  const workspaceId = getResourceId(workspaceName, workspaceNamePrefix);
  return workspaceId && userEmail
    ? `${AGENT_PREFERENCES_KEY}.${workspaceId.length}:${workspaceId}.${userEmail}`
    : null;
};

export const normalizeAgentPreferences = (value: unknown): AgentPreferences => {
  const raw =
    typeof value === "object" && value !== null && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : {};
  return {
    sendWithEnter:
      typeof raw.sendWithEnter === "boolean"
        ? raw.sendWithEnter
        : DEFAULT_AGENT_PREFERENCES.sendWithEnter,
    tokenLimitPerRun:
      typeof raw.tokenLimitPerRun === "number" &&
      Number.isSafeInteger(raw.tokenLimitPerRun) &&
      raw.tokenLimitPerRun > 0
        ? raw.tokenLimitPerRun
        : DEFAULT_AGENT_PREFERENCES.tokenLimitPerRun,
    openingBehavior:
      raw.openingBehavior === "landing" || raw.openingBehavior === "continue"
        ? raw.openingBehavior
        : DEFAULT_AGENT_PREFERENCES.openingBehavior,
  };
};
