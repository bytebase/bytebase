import { Check, EyeOff, ScrollText, X } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import type { MCPInfo } from "@/types/proto-es/v1/workspace_service_pb";

interface Line {
  readonly key: string;
  readonly icon: ReactNode;
  readonly text: string;
}

interface Props {
  readonly info: MCPInfo;
}

/**
 * What the workspace's ceiling lets this session do, shown before the person
 * approves rather than after.
 *
 * The same ceiling refuses the POST server-side, so this page is the richer
 * render of a decision the backend makes either way — never the decision
 * itself.
 */
export function MCPConsentCeiling({ info }: Props) {
  const { t } = useTranslation();

  const readWrite = info.capability === MCPSetting_Capability.READ_WRITE;
  const modeKey = readWrite ? "read-write" : "read-only";
  const modeLabel = t(`settings.mcp.policy.mode.${modeKey}.title`);

  const lines: Line[] = [
    {
      key: "read",
      icon: <Check className="size-4 text-success" />,
      text: t("oauth2.consent.mcp.line.read"),
    },
    readWrite
      ? {
          key: "write",
          icon: <Check className="size-4 text-success" />,
          text: t("oauth2.consent.mcp.line.write"),
        }
      : {
          key: "no-write",
          icon: <X className="size-4 text-error" />,
          text: t("oauth2.consent.mcp.line.no-write"),
        },
    // The WRITE class is not only data and schema: CreateIssue, CreatePlan,
    // CreateRollout, BatchRunTasks, Export and the saved-query methods all
    // carry it, so approving read-write approves workflow and egress too.
    ...(readWrite
      ? [
          {
            key: "workflow",
            icon: <Check className="size-4 text-success" />,
            text: t("oauth2.consent.mcp.line.workflow"),
          },
        ]
      : []),
    {
      key: "capped",
      icon: <Check className="size-4 text-success" />,
      text: t("oauth2.consent.mcp.line.capped"),
    },
    // Both halves, because the line promises a restriction. The toggle
    // withholds unmasking exemptions from MCP sessions, which changes nothing
    // where masking does not run at all — asserting it there would tell the
    // person approving that their data is covered when it is not.
    ...(info.ignoreMaskingExemptions && info.dataMaskingAvailable
      ? [
          {
            key: "masking",
            icon: <EyeOff className="size-4 text-control-light" />,
            text: t("oauth2.consent.mcp.line.masking"),
          },
        ]
      : []),
    {
      key: "audit",
      icon: <ScrollText className="size-4 text-control-light" />,
      text: t("oauth2.consent.mcp.line.audit"),
    },
  ];

  return (
    <div className="flex flex-col gap-3">
      <div className="bg-control-bg rounded-sm p-4 flex flex-col gap-3">
        <div className="flex items-center justify-between gap-x-2">
          <p className="text-sm text-control-light">
            {t("oauth2.consent.mcp.title")}
          </p>
          <Badge variant={readWrite ? "warning" : "success"}>{modeLabel}</Badge>
        </div>
        <ul className="text-sm text-main flex flex-col gap-2">
          {lines.map((line) => (
            <li key={line.key} className="flex items-start gap-2">
              <span className="mt-0.5 shrink-0">{line.icon}</span>
              <span>{line.text}</span>
            </li>
          ))}
        </ul>
      </div>

      {readWrite && (
        <Alert
          variant="warning"
          description={t("oauth2.consent.mcp.write-caution")}
        />
      )}
    </div>
  );
}
