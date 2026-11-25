import { ExternalLink } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Button, buttonVariants } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui/tabs";
import { useServerState } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";

export const MCP_SETUP_GUIDE_URL = "https://docs.bytebase.com/integrations/mcp";
const DESKTOP_GUIDE_URL = `${MCP_SETUP_GUIDE_URL}#claude-desktop`;
const VSCODE_GUIDE_URL = `${MCP_SETUP_GUIDE_URL}#vs-code`;

const EXTERNAL_URL_PLACEHOLDER = "{https://your-bytebase-url.com}";

// Claude Desktop's own labels and file name, shown as that app shows them.
const DESKTOP_SETTINGS_PATH =
  "Settings > Desktop app > Extensions > Edit config";
const DESKTOP_CONFIG_FILE = "claude_desktop_config.json";
const DESKTOP_TOOL_ID = "claude-desktop";

// Text the user copies: a command for a terminal, or an entry for a file.
interface CodeSpec {
  // Where the text goes.
  readonly title: string;
  readonly value: string;
  // A command wraps; a config keeps its shape and scrolls sideways.
  readonly wrap: boolean;
}

// A link that opens the tool with this server filled in. The page cannot tell
// whether the tool is installed, so the guide beside it gives the other way.
interface InstallSpec {
  readonly label: string;
  readonly href: string;
  readonly guideLabel: string;
  readonly guideHref: string;
}

// A tool is set up either by copying `block` or by following `install`.
interface Tool {
  readonly id: string;
  readonly title: string;
  readonly signIn: string;
  readonly block?: CodeSpec;
  readonly install?: InstallSpec;
  readonly note?: string;
}

export function MCPConnectSection() {
  const { t } = useTranslation();
  const { externalUrl, needConfigureExternalUrl } = useServerState();
  // Every value in step 1 carries the External URL. Without one it shows a
  // placeholder, and a tool set up with a placeholder cannot reach Bytebase,
  // so copy stays off until the URL is set.
  const urlReady = !needConfigureExternalUrl && !!externalUrl;
  const base = urlReady
    ? externalUrl.replace(/\/$/, "")
    : EXTERNAL_URL_PLACEHOLDER;
  const endpoint = `${base}/mcp`;

  const tools = useMemo<Tool[]>(() => {
    const allow = t("oauth2.consent.allow-access");
    const terminal = t("settings.mcp.connect.terminal");
    const signIn = t("settings.mcp.connect.sign-in.default", { allow });
    const command = (id: string, title: string, value: string): Tool => ({
      id,
      title,
      block: { title: terminal, value, wrap: true },
      signIn,
    });
    return [
      {
        ...command(
          "claude-code",
          "Claude Code",
          `claude mcp add --transport http bytebase ${endpoint}`
        ),
        signIn: t("settings.mcp.connect.sign-in.claude-code", { allow }),
      },
      {
        id: DESKTOP_TOOL_ID,
        title: "Claude Desktop",
        // claude_desktop_config.json takes only servers Claude starts with a
        // command, so the entry starts the npm package, which forwards to
        // this instance. Hand-formatted to keep args on one line.
        block: {
          title: DESKTOP_CONFIG_FILE,
          value: [
            '"mcpServers": {',
            '  "bytebase": {',
            '    "command": "npx",',
            '    "args": ["-y", "@bytebase/bytebase-mcp"],',
            `    "env": { "BYTEBASE_URL": "${base}" }`,
            "  }",
            "}",
          ].join("\n"),
          wrap: false,
        },
        signIn: t("settings.mcp.connect.sign-in.claude-desktop", { allow }),
      },
      command("codex", "Codex", `codex mcp add bytebase --url ${endpoint}`),
      command(
        "copilot-cli",
        "Copilot CLI",
        `copilot mcp add --transport http bytebase ${endpoint}`
      ),
      command(
        "gemini-cli",
        "Gemini CLI",
        `gemini mcp add --transport http bytebase ${endpoint}`
      ),
      {
        id: "vscode",
        title: "VS Code",
        signIn,
        install: {
          label: t("settings.mcp.connect.vscode.install"),
          href: `vscode:mcp/install?${encodeURIComponent(
            JSON.stringify({ name: "bytebase", type: "http", url: endpoint })
          )}`,
          guideLabel: t("settings.mcp.connect.vscode.guide"),
          guideHref: VSCODE_GUIDE_URL,
        },
      },
      {
        id: "other",
        title: t("settings.mcp.connect.other.tab"),
        block: {
          title: t("settings.mcp.connect.other.title"),
          value: JSON.stringify(
            { mcpServers: { bytebase: { type: "http", url: endpoint } } },
            null,
            2
          ),
          wrap: false,
        },
        signIn,
        note: t("settings.mcp.connect.other.note"),
      },
    ];
  }, [base, endpoint, t]);

  const [toolId, setToolId] = useState("claude-code");
  const selected = tools.find((tool) => tool.id === toolId) ?? tools[0];
  const firstPrompt = t("settings.mcp.first-prompt.example");

  return (
    <div className="flex flex-col gap-y-3">
      <div className="flex items-center justify-between gap-x-2">
        <h3 className="text-base font-medium">
          {t("settings.mcp.connect.title")}
        </h3>
        <GuideLink href={MCP_SETUP_GUIDE_URL}>
          {t("settings.mcp.connect.setup-guide")}
        </GuideLink>
      </div>
      <ol className="flex max-w-3xl flex-col">
        <Step index={1} title={t("settings.mcp.connect.step.add")}>
          <Tabs
            value={toolId}
            onValueChange={(value) => setToolId(String(value))}
          >
            <TabsList className="overflow-x-auto overflow-y-hidden">
              {tools.map((tool) => (
                <TabsTrigger key={tool.id} value={tool.id} className="shrink-0">
                  {tool.title}
                </TabsTrigger>
              ))}
            </TabsList>
            {tools.map((tool) => (
              <TabsPanel key={tool.id} value={tool.id}>
                <ToolPanel tool={tool} urlReady={urlReady} />
              </TabsPanel>
            ))}
          </Tabs>
        </Step>
        <Step index={2} title={t("settings.mcp.connect.step.sign-in")}>
          <p className="max-w-prose text-sm text-pretty text-control">
            {selected.signIn}
          </p>
        </Step>
        <Step index={3} title={t("settings.mcp.connect.step.prompt")} last>
          <CodeBlock
            title={t("settings.mcp.connect.prompt-title")}
            value={firstPrompt}
            copyLabel={t("settings.mcp.connect.copy-prompt")}
            wrap
          />
        </Step>
      </ol>
    </div>
  );
}

function Step({
  index,
  title,
  last,
  children,
}: {
  readonly index: number;
  readonly title: string;
  readonly last?: boolean;
  readonly children: ReactNode;
}) {
  return (
    <li className="flex gap-x-3">
      <div className="flex flex-col items-center gap-y-2">
        <span
          aria-hidden="true"
          className="flex size-6 shrink-0 items-center justify-center rounded-full bg-control-bg text-xs font-medium text-control tabular-nums"
        >
          {index}
        </span>
        {!last && <span className="w-px flex-1 bg-block-border" />}
      </div>
      <div className={cn("min-w-0 flex-1", !last && "pb-6")}>
        <h4 className="text-sm leading-6 font-medium text-main">{title}</h4>
        <div className="mt-2">{children}</div>
      </div>
    </li>
  );
}

function ToolPanel({
  tool,
  urlReady,
}: {
  readonly tool: Tool;
  readonly urlReady: boolean;
}) {
  if (tool.install) {
    return <InstallSetup install={tool.install} urlReady={urlReady} />;
  }
  if (!tool.block) return null;
  if (tool.id === DESKTOP_TOOL_ID) {
    return (
      <DesktopSetup
        toolTitle={tool.title}
        block={tool.block}
        urlReady={urlReady}
      />
    );
  }
  return (
    <div className="flex flex-col gap-y-2">
      <ToolBlock
        toolTitle={tool.title}
        block={tool.block}
        urlReady={urlReady}
      />
      {urlReady && tool.note && (
        <p className="text-sm text-control-light">{tool.note}</p>
      )}
    </div>
  );
}

interface BlockProps {
  readonly toolTitle: string;
  readonly block: CodeSpec;
  readonly urlReady: boolean;
}

function ToolBlock({ toolTitle, block, urlReady }: BlockProps) {
  const { t } = useTranslation();
  return (
    <>
      <CodeBlock
        title={block.title}
        value={block.value}
        copyLabel={t("settings.mcp.connect.copy-setup", { tool: toolTitle })}
        copyOff={!urlReady}
        wrap={block.wrap}
      />
      {/* Visible text, not a tooltip: it is the only thing that says why copy is off. */}
      {!urlReady && (
        <p className="text-sm text-control-light">
          {t("settings.mcp.connect.copy-off")}
        </p>
      )}
    </>
  );
}

// The link carries the External URL, so without one it is a disabled button
// and no link holding the placeholder is in the page.
function InstallSetup({
  install,
  urlReady,
}: {
  readonly install: InstallSpec;
  readonly urlReady: boolean;
}) {
  const { t } = useTranslation();
  const content = (
    <>
      <ExternalLink className="size-4" aria-hidden="true" />
      {install.label}
    </>
  );
  return (
    <div className="flex flex-col gap-y-2">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        {urlReady ? (
          <a href={install.href} className={buttonVariants()}>
            {content}
          </a>
        ) : (
          <Button disabled>{content}</Button>
        )}
        <GuideLink href={install.guideHref}>{install.guideLabel}</GuideLink>
      </div>
      {!urlReady && (
        <p className="text-sm text-control-light">
          {t("settings.mcp.connect.vscode.off")}
        </p>
      )}
    </div>
  );
}

// Claude Desktop has no install command, so its tab walks through adding the
// entry to its config file.
function DesktopSetup({ toolTitle, block, urlReady }: BlockProps) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-y-3">
      <ol className="flex list-decimal flex-col gap-y-2 pl-5 text-sm text-control marker:text-control-light">
        <li className="pl-1">
          <Trans
            i18nKey="settings.mcp.connect.desktop.open-folder"
            values={{ path: DESKTOP_SETTINGS_PATH }}
            components={{ ui: <span className="font-medium text-main" /> }}
          />
        </li>
        <li className="pl-1">
          <Trans
            i18nKey="settings.mcp.connect.desktop.open-file"
            values={{ file: DESKTOP_CONFIG_FILE }}
            components={{
              code: (
                <code className="rounded-xs bg-control-bg px-1 py-0.5 font-mono text-xs text-main" />
              ),
            }}
          />
        </li>
        <li className="pl-1">
          {t("settings.mcp.connect.desktop.add")}
          <div className="mt-2 flex flex-col gap-y-2">
            <ToolBlock
              toolTitle={toolTitle}
              block={block}
              urlReady={urlReady}
            />
          </div>
        </li>
      </ol>
      <p className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-control-light">
        {t("settings.mcp.connect.desktop.requirement")}
        <GuideLink href={DESKTOP_GUIDE_URL}>
          {t("settings.mcp.connect.desktop.example")}
        </GuideLink>
      </p>
    </div>
  );
}

// The heading says where the text goes and holds the copy button, so the
// button never sits over the text.
function CodeBlock({
  title,
  value,
  copyLabel,
  copyOff = false,
  wrap,
}: {
  readonly title: string;
  readonly value: string;
  readonly copyLabel: string;
  readonly copyOff?: boolean;
  readonly wrap: boolean;
}) {
  return (
    <div className="overflow-hidden rounded-sm border border-block-border">
      <div className="flex items-center justify-between gap-x-2 border-b border-block-border bg-background py-1 pr-1 pl-3">
        <span className="min-w-0 truncate text-xs text-control">{title}</span>
        <div className="shrink-0">
          <CopyButton
            content={value}
            size="sm"
            label={copyLabel}
            disabled={copyOff}
          />
        </div>
      </div>
      <pre
        className={cn(
          "bg-control-bg p-3 font-mono text-sm leading-5 text-main",
          wrap ? "wrap-anywhere whitespace-pre-wrap" : "overflow-x-auto"
        )}
      >
        {value}
      </pre>
    </div>
  );
}

function GuideLink({
  href,
  children,
}: {
  readonly href: string;
  readonly children: ReactNode;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex shrink-0 items-center gap-x-1 text-sm text-accent hover:underline"
    >
      {children}
      <ExternalLink className="size-3" aria-hidden="true" />
    </a>
  );
}
