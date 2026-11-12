import type { DeniedQueryRequestDraft } from "@/modules/sql-editor/model/accessGrantDraft";
import type { AgentMaskingPageResource } from "./chatContext";
import type { MaskingSkillSection } from "./skills";

export interface ToolDefinition {
  name: string;
  description: string;
  parametersSchema: Record<string, unknown>;
}

export interface ToolCall {
  id: string;
  name: string;
  arguments: string; // JSON-encoded
  metadata?: string; // Opaque provider-specific data (e.g., Gemini thought_signature)
}

export interface Message {
  role: "system" | "user" | "assistant" | "tool";
  content?: string;
  toolCalls?: ToolCall[];
  toolCallId?: string;
}

export type AgentChatStatus =
  | "idle"
  | "running"
  | "awaiting_user"
  | "paused"
  | "error";

export type AgentInterruptionReason = "user" | "unexpected";

export type AgentRunMode = "default" | "read-only";

export interface AgentModelCallUsage {
  attemptId?: string;
  purpose: "agent" | "follow-up";
  toolNames: string[];
  totalTokens?: number;
  inputTokens?: number;
  outputTokens?: number;
  cachedInputTokens?: number;
}

export interface AgentRunBudget {
  version: 1;
  runId: string;
  limit: number;
  totalTokensUsed: number;
  hasUnreportedUsage: boolean;
  providerAttemptCount: number;
  revision: number;
}

export interface AgentRunBindings {
  page: AgentChatSnapshot;
  context?: string;
  references: AgentResourceReference[];
  runMode: AgentRunMode;
  capability?: AgentDiagnosticCapability;
  instanceName?: string;
  maskingPage?: AgentMaskingPageResource;
}

export interface AgentRunCheckpoint {
  version: 1;
  runId: string;
  suspension: "token-limit" | "awaiting-user" | "interrupted";
  rootUserMessageId: string;
  throughMessageId: string;
  bindings: AgentRunBindings;
  maskingSections: MaskingSkillSection[];
  phase: "working" | "finalizing";
  startupComplete?: boolean;
  referencePreflightPending?: boolean;
  finalizationText?: string;
}

export interface AgentModelAttempt {
  attemptId: string;
  runId: string;
  ordinal: number;
  outcome: "response" | "error" | "aborted";
  usage?: AgentModelCallUsage;
}

export type AgentAttemptReservation =
  | { kind: "allowed"; attemptId: string; ordinal: number }
  | { kind: "token-limit" }
  | { kind: "attempt-limit" }
  | { kind: "superseded" }
  | { kind: "storage-unavailable" };

export interface AgentChatSnapshot {
  path: string;
  title: string;
}

export type AgentEvidenceReference =
  | {
      kind: "issue";
      name: string;
      title: string;
    }
  | {
      kind: "plan";
      name: string;
      title: string;
    }
  | {
      kind: "database";
      name: string;
    };

export interface AgentEvidence {
  project?: string;
  workspace?: string;
  references: AgentEvidenceReference[];
}

export type AgentAnswerEvidence = AgentEvidence | AgentEvidence[];

export type AgentHandoff =
  | { kind: "open-global-masking" }
  | { kind: "open-semantic-types" }
  | { kind: "open-data-classification" }
  | { kind: "open-project-masking-exemptions"; project: string }
  | { kind: "open-database-catalog"; project: string; database: string }
  | { kind: "review-workspace-issues"; workspace: string }
  | {
      kind:
        | "review-project-issues"
        | "review-database-changes"
        | "open-project-instances"
        | "transfer-project-databases"
        | "connect-project-instance"
        | "create-project-database";
      project: string;
    }
  | { kind: "view-failed-tasks"; plan: string }
  | { kind: "open-issue"; issue: string }
  | { kind: "review-issue-plan"; plan: string }
  | { kind: "view-failed-task"; task: string };

export type AgentResponseAction =
  | { kind: "open"; target: AgentHandoff }
  | { kind: "draft-plan"; project: string; database: string }
  | { kind: "draft-issue-comment"; issue: string; comment: string }
  | { kind: "draft-masking-rule"; draftId: string }
  | { kind: "draft-masking-exemption"; draftId: string }
  | { kind: "draft-semantic-type"; draftId: string }
  | { kind: "continue"; prompt: string };

export interface AgentResourceReference {
  kind: "project" | "database" | "issue" | "plan";
  name: string;
  project: string;
  label: string;
}

export type AgentDiagnosticCapability = {
  kind: "diagnostic";
  resource: AgentResourceReference;
};

export interface AgentMentionRange {
  start: number;
  end: number;
}

export interface AgentResourceMention {
  resource: AgentResourceReference;
  range: AgentMentionRange;
}

export interface AgentDraft {
  text: string;
  mentions?: AgentResourceMention[];
  project?: {
    name: string;
    title: string;
  };
  resource?: AgentResourceReference;
  mention?: AgentMentionRange;
}

export interface AgentMessageMetadata {
  runPreflight?: true;
  route?: string;
  pending?: boolean;
  hidden?: boolean;
  error?: string;
  runId?: string;
  interruptionReason?: AgentInterruptionReason;
  evidence?: AgentAnswerEvidence;
  handoff?: AgentHandoff;
  actions?: AgentResponseAction[];
  maskingRuleDraftHandoff?: {
    draftId: string;
    outcome: "opened" | "applied";
  };
  maskingExemptionDraftHandoff?: {
    draftId: string;
    outcome: "opened" | "applied";
    created?: true;
  };
  accessRequestDraft?: DeniedQueryRequestDraft;
  semanticTypeDraftHandoff?: {
    draftId: string;
    outcome: "opened";
    created?: true;
  };
  resourceReference?: AgentResourceReference;
  resourceReferences?: AgentResourceReference[];
}

export interface AgentMessage extends Message {
  id: string;
  chatId: string;
  createdTs: number;
  metadata?: AgentMessageMetadata;
}

export interface AgentChat {
  id: string;
  title: string;
  context?: string;
  workspaceName?: string;
  createdTs: number;
  updatedTs: number;
  status: AgentChatStatus;
  totalTokensUsed: number;
  runBudget?: AgentRunBudget;
  runCheckpoint?: AgentRunCheckpoint;
  modelAttempts?: AgentModelAttempt[];
  legacyUsage?: { totalTokensUsed: number; hasUnreportedUsage: boolean };
  lastRunUsage?: {
    runId: string;
    calls: AgentModelCallUsage[];
  };
  hasUnreportedUsage?: boolean;
  page?: AgentChatSnapshot;
  archived: boolean;
  lastError?: string | null;
  requiresAIConfiguration?: boolean;
  interrupted?: boolean;
  interruptionReason?: AgentInterruptionReason;
  runId?: string | null;
  runMode?: AgentRunMode;
  capability?: AgentDiagnosticCapability;
}

export type AgentAskUserKind = "input" | "confirm" | "choose";

export interface AgentAskUserOption {
  label: string;
  value: string;
  description?: string;
}

export interface AgentPendingAsk {
  toolCallId: string;
  prompt: string;
  kind: AgentAskUserKind;
  apiPermission?: {
    actionId: string;
    operationId: string;
  };
  defaultValue?: string;
  defaultResource?: AgentResourceReference;
  confirmLabel?: string;
  cancelLabel?: string;
  options?: AgentAskUserOption[];
}

export type AgentAskUserResponse =
  | {
      kind: "input";
      answer: string;
    }
  | {
      kind: "confirm";
      answer: string;
      confirmed: boolean;
    }
  | {
      kind: "choose";
      answer: string;
      value: string;
    };

export type ToolExecutionResult =
  | {
      kind: "tool_result";
      result: string;
    }
  | {
      kind: "done";
      text: string;
      success: boolean;
      handoff?: AgentHandoff;
      actions?: AgentResponseAction[];
    }
  | {
      kind: "ask_user";
      ask: AgentPendingAsk;
    };

interface AgentLoopUsage {
  totalTokensUsed?: number;
}

export type AgentLoopOutcome =
  | ({
      kind: "stopped";
      reason: "provider-attempt-limit";
    } & AgentLoopUsage)
  | ({
      kind: "paused";
      reason: "token-limit";
      phase: "working" | "finalizing";
      finalizationText?: string;
    } & AgentLoopUsage)
  | ({
      kind: "completed";
      text: string;
      success: boolean;
      handoff?: AgentHandoff;
      actions?: AgentResponseAction[];
    } & AgentLoopUsage)
  | ({
      kind: "awaiting_user";
      ask: AgentPendingAsk;
    } & AgentLoopUsage)
  | ({
      kind: "aborted";
    } & AgentLoopUsage)
  | ({
      kind: "error";
      error: Error;
    } & AgentLoopUsage);

export type ToolExecutor = (
  name: string,
  args: Record<string, unknown>,
  toolCallId: string
) => Promise<ToolExecutionResult>;
