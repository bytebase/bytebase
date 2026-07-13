import { ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";

export type ConnectionFailureCategory =
  | "auth_failed"
  | "network_unreachable"
  | "permission_denied"
  | "ssl_tls_failed"
  | "timeout"
  | "unsupported_engine"
  | "unknown";

const connectionFailureCategorySet = new Set<ConnectionFailureCategory>([
  "auth_failed",
  "network_unreachable",
  "permission_denied",
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

export function ConnectionRecovery({
  category,
  className,
}: Readonly<{
  category: ConnectionFailureCategory;
  className?: string;
}>) {
  const { t } = useTranslation();
  const recovery = {
    auth_failed: {
      title: t("instance.connection-recovery.auth.title"),
      description: t("instance.connection-recovery.auth.description"),
      steps: [
        t("instance.connection-recovery.auth.steps.credentials"),
        t("instance.connection-recovery.auth.steps.method"),
        t("instance.connection-recovery.auth.steps.retry"),
      ],
    },
    network_unreachable: {
      title: t("instance.connection-recovery.network.title"),
      description: t("instance.connection-recovery.network.description"),
      steps: [
        t("instance.connection-recovery.network.steps.address"),
        t("instance.connection-recovery.network.steps.firewall"),
        t("instance.connection-recovery.network.steps.private-network"),
      ],
    },
    permission_denied: {
      title: t("instance.connection-recovery.permission.title"),
      description: t("instance.connection-recovery.permission.description"),
      steps: [
        t("instance.connection-recovery.permission.steps.account"),
        t("instance.connection-recovery.permission.steps.database"),
        t("instance.connection-recovery.permission.steps.workflow"),
      ],
    },
    timeout: {
      title: t("instance.connection-recovery.timeout.title"),
      description: t("instance.connection-recovery.timeout.description"),
      steps: [
        t("instance.connection-recovery.timeout.steps.load"),
        t("instance.connection-recovery.timeout.steps.retry"),
        t("instance.connection-recovery.timeout.steps.route"),
      ],
    },
    ssl_tls_failed: {
      title: t("instance.connection-recovery.tls.title"),
      description: t("instance.connection-recovery.tls.description"),
      steps: [
        t("instance.connection-recovery.tls.steps.certificates"),
        t("instance.connection-recovery.tls.steps.hostname"),
        t("instance.connection-recovery.tls.steps.mode"),
      ],
    },
    unsupported_engine: {
      title: t("instance.connection-recovery.unsupported.title"),
      description: t("instance.connection-recovery.unsupported.description"),
      steps: [
        t("instance.connection-recovery.unsupported.steps.engine"),
        t("instance.connection-recovery.unsupported.steps.retry"),
        t("instance.connection-recovery.unsupported.steps.version"),
      ],
    },
    unknown: {
      title: t("instance.connection-recovery.unknown.title"),
      description: t("instance.connection-recovery.unknown.description"),
      steps: [
        t("instance.connection-recovery.unknown.steps.fields"),
        t("instance.connection-recovery.unknown.steps.help"),
        t("instance.connection-recovery.unknown.steps.logs"),
      ],
    },
  }[category];

  return (
    <Alert
      variant="warning"
      title={recovery.title}
      description={recovery.description}
      className={className}
    >
      <ul className="mt-2 list-disc pl-5 text-sm leading-6 opacity-90">
        {recovery.steps.map((step) => (
          <li key={step}>{step}</li>
        ))}
      </ul>
      <a
        className="mt-3 inline-flex items-center gap-x-1 text-sm font-medium underline underline-offset-2"
        href="https://docs.bytebase.com/get-started/connect/overview?source=console"
        target="_blank"
        rel="noreferrer"
      >
        {t("instance.connection-recovery.docs")}
        <ExternalLink className="size-3.5" />
      </a>
    </Alert>
  );
}
