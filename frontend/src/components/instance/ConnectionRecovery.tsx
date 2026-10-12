import { ExternalLink } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useAppStore } from "@/stores/app";
import {
  CloudEgressAddresses,
  useCloudEgressAddresses,
} from "./CloudEgressAddresses";

export type ConnectionFailureCategory =
  | "auth_failed"
  | "host_not_found"
  | "network_unreachable"
  | "permission_denied"
  | "private_address"
  | "ssl_tls_failed"
  | "timeout"
  | "unsupported_engine"
  | "unknown";

const connectionFailureCategorySet = new Set<ConnectionFailureCategory>([
  "auth_failed",
  "host_not_found",
  "network_unreachable",
  "permission_denied",
  "private_address",
  "ssl_tls_failed",
  "timeout",
  "unsupported_engine",
  "unknown",
]);

export const connectionFailureCategoryHeader =
  "bytebase-connection-failure-category";

export const normalizeConnectionFailureCategory = (
  value: string | null | undefined
): ConnectionFailureCategory => {
  if (!value) return "unknown";
  if (connectionFailureCategorySet.has(value as ConnectionFailureCategory)) {
    return value as ConnectionFailureCategory;
  }
  return "unknown";
};

// A fix the form can apply for the user. `note` states what the action gives
// up, when it gives something up.
export type ConnectionRecoveryAction = {
  key: string;
  label: string;
  note?: string;
  disabled?: boolean;
  onClick: () => void;
};

export function ConnectionRecovery({
  category,
  host = "",
  actions = [],
  className,
}: Readonly<{
  category: ConnectionFailureCategory;
  host?: string;
  actions?: ConnectionRecoveryAction[];
  className?: string;
}>) {
  const { t } = useTranslation();
  const isSaaSMode = useAppStore((s) => s.isSaaSMode());
  const egressAddresses = useCloudEgressAddresses();
  const firewallStep: ReactNode = !isSaaSMode ? (
    t("instance.connection-recovery.network.steps.firewall-self-hosted")
  ) : egressAddresses.length > 0 ? (
    <span className="inline-flex flex-wrap items-center gap-x-2 gap-y-1">
      {t("instance.sentence.firewall-info-addresses")}
      <CloudEgressAddresses addresses={egressAddresses} />
    </span>
  ) : (
    t("instance.connection-recovery.network.steps.firewall-saas")
  );
  const recoveries: Record<
    ConnectionFailureCategory,
    {
      title: string;
      description: string;
      steps: { key: string; content: ReactNode }[];
    }
  > = {
    auth_failed: {
      title: t("instance.connection-recovery.auth.title"),
      description: t("instance.connection-recovery.auth.description"),
      steps: [
        {
          key: "credentials",
          content: t("instance.connection-recovery.auth.steps.credentials"),
        },
        {
          key: "method",
          content: t("instance.connection-recovery.auth.steps.method"),
        },
        {
          key: "retry",
          content: t("instance.connection-recovery.auth.steps.retry"),
        },
      ],
    },
    host_not_found: {
      title: t("instance.connection-recovery.host-not-found.title"),
      description: isSaaSMode
        ? t("instance.connection-recovery.host-not-found.description-saas")
        : t(
            "instance.connection-recovery.host-not-found.description-self-hosted"
          ),
      steps: [],
    },
    network_unreachable: {
      title: t("instance.connection-recovery.network.title"),
      description: isSaaSMode
        ? t("instance.connection-recovery.network.description-saas")
        : t("instance.connection-recovery.network.description-self-hosted"),
      steps: [
        {
          key: "address",
          content: t("instance.connection-recovery.network.steps.address"),
        },
        { key: "firewall", content: firewallStep },
        {
          key: "private-network",
          content: t(
            "instance.connection-recovery.network.steps.private-network"
          ),
        },
      ],
    },
    permission_denied: {
      title: t("instance.connection-recovery.permission.title"),
      description: t("instance.connection-recovery.permission.description"),
      steps: [
        {
          key: "account",
          content: t("instance.connection-recovery.permission.steps.account"),
        },
        {
          key: "database",
          content: t("instance.connection-recovery.permission.steps.database"),
        },
        {
          key: "workflow",
          content: t("instance.connection-recovery.permission.steps.workflow"),
        },
      ],
    },
    private_address: {
      title: t("instance.connection-recovery.private.title"),
      description: t("instance.connection-recovery.private.description", {
        host,
      }),
      steps: [],
    },
    timeout: {
      title: t("instance.connection-recovery.timeout.title"),
      description: isSaaSMode
        ? t("instance.connection-recovery.timeout.description-saas")
        : t("instance.connection-recovery.timeout.description-self-hosted"),
      steps: [
        ...(isSaaSMode && egressAddresses.length > 0
          ? [{ key: "firewall", content: firewallStep }]
          : []),
        {
          key: "load",
          content: t("instance.connection-recovery.timeout.steps.load"),
        },
        {
          key: "retry",
          content: t("instance.connection-recovery.timeout.steps.retry"),
        },
        {
          key: "route",
          content: t("instance.connection-recovery.timeout.steps.route"),
        },
      ],
    },
    ssl_tls_failed: {
      title: t("instance.connection-recovery.tls.title"),
      description: t("instance.connection-recovery.tls.description"),
      steps: [
        {
          key: "certificates",
          content: t("instance.connection-recovery.tls.steps.certificates"),
        },
        {
          key: "hostname",
          content: t("instance.connection-recovery.tls.steps.hostname"),
        },
        {
          key: "mode",
          content: t("instance.connection-recovery.tls.steps.mode"),
        },
      ],
    },
    unsupported_engine: {
      title: t("instance.connection-recovery.unsupported.title"),
      description: t("instance.connection-recovery.unsupported.description"),
      steps: [
        {
          key: "engine",
          content: t("instance.connection-recovery.unsupported.steps.engine"),
        },
        {
          key: "retry",
          content: t("instance.connection-recovery.unsupported.steps.retry"),
        },
        {
          key: "version",
          content: t("instance.connection-recovery.unsupported.steps.version"),
        },
      ],
    },
    unknown: {
      title: t("instance.connection-recovery.unknown.title"),
      description: t("instance.connection-recovery.unknown.description"),
      steps: [
        {
          key: "fields",
          content: t("instance.connection-recovery.unknown.steps.fields"),
        },
        {
          key: "help",
          content: t("instance.connection-recovery.unknown.steps.help"),
        },
        {
          key: "logs",
          content: isSaaSMode
            ? t("instance.connection-recovery.unknown.steps.logs-saas")
            : t("instance.connection-recovery.unknown.steps.logs-self-hosted"),
        },
      ],
    },
  };
  const recovery = recoveries[category];
  const notes = actions.filter((action) => action.note);

  return (
    <Alert
      variant="warning"
      title={recovery.title}
      description={recovery.description}
      className={className}
    >
      {recovery.steps.length > 0 && (
        <ul className="mt-2 list-disc pl-5 text-sm leading-6 opacity-90">
          {recovery.steps.map((step) => (
            <li key={step.key}>{step.content}</li>
          ))}
        </ul>
      )}
      {actions.length > 0 && (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          {actions.map((action) => (
            <Button
              key={action.key}
              size="sm"
              appearance="outline"
              disabled={action.disabled}
              onClick={action.onClick}
            >
              {action.label}
            </Button>
          ))}
        </div>
      )}
      {notes.map((action) => (
        <p key={action.key} className="mt-2 text-xs leading-4 opacity-90">
          {action.note}
        </p>
      ))}
      <a
        className="mt-3 inline-flex items-center gap-x-1 text-sm font-medium underline underline-offset-2"
        href={
          isSaaSMode
            ? "https://docs.bytebase.com/get-started/cloud#prerequisites"
            : "https://docs.bytebase.com/get-started/connect/overview?source=console"
        }
        target="_blank"
        rel="noreferrer"
      >
        {t("instance.connection-recovery.docs")}
        <ExternalLink className="size-3.5" />
      </a>
    </Alert>
  );
}
