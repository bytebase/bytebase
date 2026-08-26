import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { ExternalUrlAlert } from "@/components/ExternalUrlAlert";
import { Alert } from "@/components/ui/alert";
import { CopyButton } from "@/components/ui/copy-button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useServerState } from "@/hooks/useAppState";
import { isDev } from "@/utils";
import { MCPAccessPolicySection } from "./mcp/MCPAccessPolicySection";

export function MCPPage() {
  const { t } = useTranslation();
  const { externalUrl, needConfigureExternalUrl } = useServerState();

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

  // One tab per client, each carrying the whole of what that client needs:
  // a command where the client has a CLI, the config object where it does not.
  const tabs = useMemo(
    () => [
      {
        id: "claude-code",
        title: "Claude Code",
        command: `claude mcp add --transport http bytebase ${mcpEndpointUrl}`,
      },
      {
        id: "claude-desktop",
        // The URL, not the config object: claude_desktop_config.json only
        // reaches LOCAL servers. A remote one is added as a custom connector
        // on claude.ai and shows up in the desktop app from there.
        title: "Claude Desktop",
        url: mcpEndpointUrl,
        note: t("settings.mcp.connect.desktop-note"),
      },
      {
        id: "codex",
        title: "Codex",
        command: `codex mcp add bytebase --url ${mcpEndpointUrl}`,
      },
      {
        id: "copilot-cli",
        title: "Copilot CLI",
        command: `copilot mcp add --transport http bytebase ${mcpEndpointUrl}`,
      },
      {
        id: "gemini-cli",
        title: "Gemini CLI",
        command: `gemini mcp add --transport http bytebase ${mcpEndpointUrl}`,
      },
      {
        id: "vscode",
        title: "VS Code",
        command: `code --add-mcp '{"name":"bytebase","type":"http","url":"${mcpEndpointUrl}"}'`,
      },
      {
        id: "json",
        title: "JSON",
        config: generalConfig,
        note: t("settings.mcp.connect.json-note"),
      },
    ],
    [mcpEndpointUrl, generalConfig, t]
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
      <ExternalUrlAlert actionAppearance="outline" />

      {/* Hidden in prod builds until release readiness; the settings API and
          server-side enforcement stay live intentionally. */}
      {isDev() && <MCPAccessPolicySection />}

      {/* Authentication Notice */}
      <Alert
        variant="info"
        title={t("settings.mcp.auth.title")}
        description={t("settings.mcp.auth.description")}
      />

      {/* Connect a client */}
      <div className="flex flex-col gap-y-3">
        <div className="flex flex-col gap-y-1">
          <h3 className="text-base font-medium">
            {t("settings.mcp.connect.title")}
          </h3>
          <p className="text-sm text-control-light">
            {t("settings.mcp.connect.description")}
          </p>
        </div>
        <Tabs defaultValue="claude-code">
          <TabsList className="overflow-x-auto">
            {tabs.map((tab) => (
              <TabsTrigger key={tab.id} value={tab.id} className="shrink-0">
                {tab.title}
              </TabsTrigger>
            ))}
          </TabsList>
          {tabs.map((tab) => (
            <TabsPanel key={tab.id} value={tab.id}>
              <div className="flex flex-col gap-y-2">
                {(tab.command ?? tab.url) ? (
                  <div className="flex items-center gap-x-2">
                    <Input
                      readOnly
                      value={tab.command ?? tab.url}
                      className="flex-1 font-mono"
                      onClick={(e) => (e.target as HTMLInputElement).select()}
                    />
                    <CopyButton
                      content={tab.command ?? tab.url ?? ""}
                      size="sm"
                    />
                  </div>
                ) : (
                  <div className="flex items-start gap-x-2">
                    <Textarea
                      readOnly
                      value={tab.config}
                      rows={7}
                      className="flex-1 font-mono text-sm"
                    />
                    <CopyButton content={tab.config ?? ""} size="sm" />
                  </div>
                )}
                {tab.note && (
                  <p className="text-sm text-control-light">{tab.note}</p>
                )}
              </div>
            </TabsPanel>
          ))}
        </Tabs>
      </div>

      {/* Your First Prompt */}
      <div className="flex flex-col gap-y-3">
        <div className="flex flex-col gap-y-1">
          <h3 className="text-base font-medium">
            {t("settings.mcp.first-prompt.title")}
          </h3>
          <p className="text-sm text-control-light">
            {t("settings.mcp.first-prompt.description")}
          </p>
        </div>
        <div className="flex items-center gap-x-2">
          <Input
            readOnly
            value={firstPromptExample}
            className="flex-1 font-mono"
            onClick={(e) => (e.target as HTMLInputElement).select()}
          />
          <CopyButton content={firstPromptExample} size="sm" />
        </div>
      </div>
    </div>
  );
}
