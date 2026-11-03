import { router } from "@/app/router";
import { useAppStore } from "@/stores/app";
import { useAgentStore } from "../store/agent";
import { getCurrentChatContext } from "./chatContext";

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

export function buildInstanceConnectionHelpPrompt(options: {
  engine: string;
  stage: "connection test" | "instance creation";
  failureCategory?: InstanceConnectionFailureCategory;
  failureCode?: string;
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

  return [
    `I'm creating a ${options.engine} instance. The ${options.stage} failed: ${summary}.${failureDetails}`,
    "The exact error text and connection settings have not been shared.",
    "Help me figure out what to check next, one step at a time. Ask for missing non-secret details instead of guessing.",
    "Never ask me to share passwords, tokens, private keys, certificate contents, or full connection strings. Do not run another connection test or submit the instance form.",
  ].join(" ");
}

export function openAgentWithPrompt(text: string): void {
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
  store.queueUserMessage({
    chatId: chat.id,
    text,
    page,
    runMode: "read-only",
  });
  store.open();
}
