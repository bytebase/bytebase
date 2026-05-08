import { Check, Copy } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Textarea } from "@/react/components/ui/textarea";
import { useServerState } from "@/react/hooks/useAppState";
import { router } from "@/router";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { hasWorkspacePermissionV2 } from "@/utils";

export function MCPPage() {
  const { t } = useTranslation();
  const { externalUrl, needConfigureExternalUrl } = useServerState();
  const canConfigureExternalUrl = hasWorkspacePermissionV2(
    "bb.settings.setWorkspaceProfile"
  );

  const mcpEndpointUrl = useMemo(() => {
    if (needConfigureExternalUrl || !externalUrl) {
      return "{https://your-bytebase-url.com}/mcp";
    }
    const base = externalUrl.replace(/\/$/, "");
    return `${base}/mcp`;
  }, [externalUrl, needConfigureExternalUrl]);

  const generalConfig = useMemo(() => {
    return JSON.stringify(
      {
        mcpServers: {
          bytebase: {
            type: "http",
            url: mcpEndpointUrl,
          },
        },
      },
      null,
      2
    );
  }, [mcpEndpointUrl]);

  const tabs = useMemo(
    () => [
      {
        id: "claude-code",
        title: "Claude Code",
        content: `claude mcp add bytebase --transport http ${mcpEndpointUrl}`,
      },
      {
        id: "codex",
        title: "Codex",
        content: `codex mcp add --name bytebase --transport http --url ${mcpEndpointUrl}`,
      },
      {
        id: "copilot-cli",
        title: "Copilot CLI",
        content: `gh copilot mcp add bytebase --transport http --url ${mcpEndpointUrl}`,
      },
      {
        id: "gemini-cli",
        title: "Gemini CLI",
        content: `gemini mcp add --name bytebase --transport http --url ${mcpEndpointUrl}`,
      },
      {
        id: "vscode",
        title: "VS Code",
        content: `code --add-mcp '{"name":"bytebase","type":"http","url":"${mcpEndpointUrl}"}'`,
      },
    ],
    [mcpEndpointUrl]
  );

  const firstPromptExample = t("settings.mcp.first-prompt.example");

  return (
    <div className="w-full px-4 flex flex-col gap-y-6 py-4">
      {/* Header */}
      <div>
        <h2 className="text-lg font-medium">{t("settings.mcp.title")}</h2>
        <p className="textinfolabel mt-1">{t("settings.mcp.description")}</p>
      </div>

      {/* Warning if external URL not configured */}
      {needConfigureExternalUrl && (
        <Alert variant="error">
          <AlertTitle>{t("banner.external-url")}</AlertTitle>
          <AlertDescription>
            <span>
              {t("settings.general.workspace.external-url.description")}
            </span>
            {canConfigureExternalUrl && (
              <Button
                variant="outline"
                size="sm"
                className="ml-3"
                onClick={() =>
                  router.push({ name: SETTING_ROUTE_WORKSPACE_GENERAL })
                }
              >
                {t("common.configure-now")}
              </Button>
            )}
          </AlertDescription>
        </Alert>
      )}

      {/* Authentication Notice */}
      <Alert variant="info">
        <AlertTitle>{t("settings.mcp.auth.title")}</AlertTitle>
        <AlertDescription>
          {t("settings.mcp.auth.description")}
        </AlertDescription>
      </Alert>

      {/* Section 1: General Config */}
      <div className="flex flex-col gap-y-3">
        <div className="flex flex-col gap-y-1">
          <div className="flex items-center gap-x-2">
            <h3 className="text-base font-medium">
              {t("settings.mcp.general-config.title")}
            </h3>
            <CopyButton content={generalConfig} />
          </div>
          <p className="text-sm text-control-light">
            {t("settings.mcp.setup.add-to-config")}
          </p>
        </div>
        <Textarea
          readOnly
          value={generalConfig}
          rows={7}
          className="font-mono text-sm"
        />
      </div>

      {/* Section 2: Quick Start CLI Commands */}
      <div className="flex flex-col gap-y-3">
        <div className="flex flex-col gap-y-1">
          <h3 className="text-base font-medium">
            {t("settings.mcp.quick-start.title")}
          </h3>
          <p className="text-sm text-control-light">
            {t("settings.mcp.quick-start.description")}
          </p>
        </div>
        <Tabs defaultValue="claude-code">
          <TabsList>
            {tabs.map((tab) => (
              <TabsTrigger key={tab.id} value={tab.id}>
                {tab.title}
              </TabsTrigger>
            ))}
          </TabsList>
          {tabs.map((tab) => (
            <TabsPanel key={tab.id} value={tab.id}>
              <div className="flex items-center gap-x-2">
                <Input
                  readOnly
                  value={tab.content}
                  className="flex-1 font-mono"
                  onClick={(e) => (e.target as HTMLInputElement).select()}
                />
                <CopyButton content={tab.content} />
              </div>
            </TabsPanel>
          ))}
        </Tabs>
      </div>

      {/* Section 3: Your First Prompt */}
      <div className="flex flex-col gap-y-3">
        <div className="flex flex-col gap-y-1">
          <div className="flex items-center gap-x-2">
            <h3 className="text-base font-medium">
              {t("settings.mcp.first-prompt.title")}
            </h3>
            <CopyButton content={firstPromptExample} />
          </div>
          <p className="text-sm text-control-light">
            {t("settings.mcp.first-prompt.description")}
          </p>
        </div>
        <code className="text-sm bg-gray-100 px-3 py-2 rounded-xs">
          {firstPromptExample}
        </code>
      </div>
    </div>
  );
}

function CopyButton({ content }: { content: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API not available
    }
  };

  return (
    <Button variant="ghost" size="sm" onClick={handleCopy}>
      {copied ? (
        <Check className="h-4 w-4 text-success" />
      ) : (
        <Copy className="h-4 w-4" />
      )}
    </Button>
  );
}
