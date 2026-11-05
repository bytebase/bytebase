import { router } from "@/app/router";
import { useAppStore } from "@/stores/app";
import { useAgentStore } from "../store/agent";
import { getCurrentChatContext } from "./chatContext";
import type { AgentResourceReference } from "./types";

export type InstanceConnectionFailureCategory =
  | "auth_failed"
  | "network_unreachable"
  | "permission_denied"
  | "ssl_tls_failed"
  | "timeout"
  | "unsupported_engine"
  | "unknown";

const connectionFailureSummary: Record<
  InstanceConnectionFailureCategory,
  string
> = {
  auth_failed: "authentication was rejected",
  network_unreachable: "the database endpoint could not be reached",
  permission_denied: "the connected account lacks required access",
  ssl_tls_failed: "TLS negotiation or certificate verification failed",
  timeout: "the connection attempt timed out",
  unsupported_engine: "the configured engine or version is unsupported",
  unknown: "the connection attempt failed for an unknown reason",
};

const MAX_FAILURE_MESSAGE_LENGTH = 2000;

function sanitizeFailureMessage(message: string): string {
  const sanitized = message
    .replace(
      /-----BEGIN [A-Z0-9 ]*(?:PRIVATE KEY|CERTIFICATE)-----[\s\S]*?-----END [A-Z0-9 ]*(?:PRIVATE KEY|CERTIFICATE)-----/gi,
      "[credential material redacted]"
    )
    .replace(/(\b[a-z][a-z0-9+.-]*:\/\/)[^/\s"'<>]*@/gi, "$1[redacted]@")
    .replace(
      /\b(?:postgres(?:ql)?|mysql|mariadb|mongodb(?:\+srv)?|mssql|sqlserver|oracle):\/\/[^\s"'<>]+/gi,
      "[connection string redacted]"
    )
    .replace(/\b(Bearer|Basic)\s+[A-Za-z0-9._~+/-]+=*/gi, "$1 [redacted]")
    .replace(
      /\b([A-Za-z0-9_-]*(?:password|passwd|pwd|token|secret|credential|private[_-]?key|api[_-]?key|access[_-]?key)[A-Za-z0-9_-]*\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,;]+)/gi,
      "$1[redacted]"
    )
    .replace(/(\b[^:/\s@]+:)[^@\s]+(@(?:tcp|unix)\()/gi, "$1[redacted]$2");

  return sanitized.length > MAX_FAILURE_MESSAGE_LENGTH
    ? `${sanitized.slice(0, MAX_FAILURE_MESSAGE_LENGTH)} [truncated]`
    : sanitized;
}

export function buildInstanceConnectionHelpPrompt(options: {
  engine: string;
  stage: "connection test" | "instance creation";
  failureCategory?: InstanceConnectionFailureCategory;
  failureCode?: string;
  failureMessage?: string;
}): string {
  const summary =
    options.stage === "connection test"
      ? connectionFailureSummary[options.failureCategory ?? "unknown"]
      : "Bytebase could not complete the create request";
  const failureDetails =
    options.stage === "connection test"
      ? ` Failure category: ${options.failureCategory ?? "unknown"}.`
      : options.failureCode
        ? ` Server status: ${options.failureCode}.`
        : "";
  const errorDetails = options.failureMessage?.trim()
    ? `Error details (common credential patterns redacted): ${sanitizeFailureMessage(options.failureMessage)}`
    : "No detailed error message was available.";

  return [
    `I'm creating a ${options.engine} instance. The ${options.stage} failed: ${summary}.${failureDetails}`,
    errorDetails,
    "Use the error details above and inspect the current page's visible non-secret connection settings. Do not ask me to paste or repeat the error.",
    "Help me figure out what to check next, one step at a time. Ask only for missing non-secret details instead of guessing.",
    "Never ask me to share passwords, tokens, private keys, certificate contents, or full connection strings. Do not run another connection test or submit the instance form.",
  ].join("\n");
}

export function openAgentWithPrompt(
  text: string,
  resourceReferences?: AgentResourceReference[],
  onChatCreated?: (chatId: string) => void
): void {
  const app = useAppStore.getState();
  const workspaceName =
    app.workspace?.name ||
    app.currentUser?.workspace ||
    app.serverInfo?.workspace ||
    undefined;
  const context = getCurrentChatContext() ?? workspaceName;
  const route = router.currentRoute.value;
  const page = {
    path: route.fullPath || window.location.pathname + window.location.hash,
    title: document.title,
  };

  const store = useAgentStore.getState();
  const chat = store.createChat({ page, context, workspaceName });
  onChatCreated?.(chat.id);
  store.queueUserMessage({
    chatId: chat.id,
    text,
    page,
    runMode: "read-only",
    ...(resourceReferences?.length ? { resourceReferences } : {}),
  });
  store.open();
}
