import { createContextValues } from "@connectrpc/connect";
import { Building2, Check, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { workspaceServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { router } from "@/app/router";
import { AUTH_SIGNIN_MODULE } from "@/app/router/handles";
import { BytebaseLogo } from "@/components/BytebaseLogo";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useWorkspace } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import type { MCPInfo } from "@/types/proto-es/v1/workspace_service_pb";
import { MCPConsentCeiling } from "./MCPConsentCeiling";
import { MCPConsentDisabled } from "./MCPConsentDisabled";

const AUTHORIZE_URL = "/api/oauth2/authorize";

export function OAuth2ConsentPage() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [clientName, setClientName] = useState("");
  // What the workspace's MCP ceiling lets this session do. Every grant this
  // server issues is an MCP grant, so the ceiling decides the POST too — this
  // is the same answer, shown before the person presses Approve.
  const [mcpInfo, setMcpInfo] = useState<MCPInfo | undefined>(undefined);

  const loadWorkspace = useAppStore((state) => state.loadWorkspace);
  const loadWorkspaceList = useAppStore((state) => state.loadWorkspaceList);
  const switchWorkspace = useAppStore((state) => state.switchWorkspace);

  const isLoggedIn = useAppStore((s) => s.isLoggedIn());
  // Workspace context shown on the consent card. On SaaS, every Bytebase
  // user belongs to at least one workspace; on self-hosted there's a single
  // implicit workspace. We display it so the user can confirm which
  // workspace this OAuth grant will be bound to.
  const isSaaSMode = useAppStore((state) => state.isSaaSMode());
  const currentWorkspace = useWorkspace();
  const workspaceList = useAppStore((state) => state.workspaceList);

  // Ensure the app store has loaded the current workspace for the consent
  // card. Idempotent: loadWorkspace returns the cached value when present.
  useEffect(() => {
    void loadWorkspace();
  }, [loadWorkspace]);

  const query = router.currentRoute.value.query;
  const clientId = (query.client_id as string) || "";
  const redirectUri = (query.redirect_uri as string) || "";
  const oauthState = (query.state as string) || "";
  const codeChallenge = (query.code_challenge as string) || "";
  const codeChallengeMethod = (query.code_challenge_method as string) || "";
  // RFC 8707 resource indicator and RFC 6749 scope, forwarded verbatim from the
  // /authorize redirect. The backend validated them there and re-validates the
  // values this page posts back, so this page only has to not drop them.
  const resource = (query.resource as string) || "";
  const scope = (query.scope as string) || "";

  const initRef = useRef(false);
  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;
    if (!isLoggedIn) {
      const returnUrl = router.currentRoute.value.fullPath;
      router.replace({
        name: AUTH_SIGNIN_MODULE,
        query: { redirect: returnUrl },
      });
      return;
    }

    if (!clientId || !redirectUri || !codeChallenge || !codeChallengeMethod) {
      setError(t("oauth2.consent.error-missing-params"));
      setLoading(false);
      return;
    }

    // This page renders outside any shell, so the workspace bootstrap hasn't
    // populated the app store — load server info so `isSaaSMode()` resolves
    // and the SaaS workspace picker can render.
    void useAppStore.getState().loadServerInfo();

    (async () => {
      try {
        const response = await fetch(
          `/api/oauth2/clients/${encodeURIComponent(clientId)}`
        );
        if (!response.ok) {
          const data = await response.json();
          setError(
            data.error_description || t("oauth2.consent.error-client-not-found")
          );
          setLoading(false);
          return;
        }
        const data = await response.json();
        setClientName(data.client_name || clientId);
      } catch {
        // Same shape as the !response.ok branch above: with no client there is
        // nothing for the policy lookup to enrich, and the render checks
        // loading before error, so falling through would hold the spinner over
        // an error we already have.
        setError(t("oauth2.consent.error-load-failed"));
        setLoading(false);
        return;
      }
      // A ceiling this build cannot read refuses this call. That is the one
      // policy state the backend still has to be the first to say, so the card
      // falls back to its ceiling-free shape and the POST refuses with its own
      // page. Silent: the interceptor's toast would be the only thing on
      // screen about it, and it names a status code, not a fix.
      try {
        setMcpInfo(
          await workspaceServiceClientConnect.getMCPInfo(
            {},
            {
              contextValues: createContextValues().set(silentContextKey, true),
              // The shared transport has no deadline. This request only
              // enriches the card, so a stalled one must not leave the page on
              // its spinner with no Allow and no Deny.
              timeoutMs: 10_000,
            }
          )
        );
      } catch {
        setMcpInfo(undefined);
      }
      setLoading(false);
    })();
  }, []);

  // Prefetch workspace list on SaaS so the picker can render. This runs in
  // its own effect keyed on `isSaaSMode` because actuator's serverInfo may
  // still be loading when the consent page first mounts; running this here
  // (instead of inside the bootstrap effect with `[]` deps) lets us pick up
  // the SaaS signal the moment it resolves true. Failure is non-fatal — the
  // current workspace is still shown without a picker.
  const prefetchRef = useRef(false);
  useEffect(() => {
    if (!isSaaSMode || prefetchRef.current) return;
    prefetchRef.current = true;
    loadWorkspaceList().catch(() => {});
  }, [isSaaSMode, loadWorkspaceList]);

  // Switch the active workspace in-place, preserving the consent flow.
  // Calls switchWorkspace with redirect=false so that
  //   (a) the bb-workspace-switch channel notification is posted on the
  //       store's own channel instance, so the store's onmessage listener
  //       in *this* tab does NOT fire (BroadcastChannel excludes the
  //       source object) — without this we'd race-redirect to the landing
  //       page and lose the OAuth query params;
  //   (b) other tabs still receive the broadcast and refresh as usual.
  // We then reload the consent URL ourselves so the session cookie carries
  // the new workspace_id into the upcoming POST /api/oauth2/authorize.
  const onSwitchWorkspace = async (workspaceName: string | null) => {
    if (!workspaceName || workspaceName === currentWorkspace?.name) return;
    setSubmitting(true);
    try {
      await switchWorkspace(workspaceName, false);
      globalThis.location.reload();
    } catch {
      setError(t("oauth2.consent.error-switch-failed"));
      setSubmitting(false);
    }
  };

  // Rendered in BOTH branches. A disabled workspace is exactly when a SaaS
  // user needs the switcher: another workspace may permit MCP, and without it
  // they have to abandon the OAuth flow, switch, and start over.
  const workspaceCard = currentWorkspace ? (
    <div className="bg-control-bg rounded-sm p-4 flex items-center gap-3">
      <Building2 className="size-5 text-control-light shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-xs text-control-light">
          {t("oauth2.consent.workspace-label")}
        </p>
        {isSaaSMode && workspaceList.length > 1 ? (
          <Select
            value={currentWorkspace.name}
            onValueChange={onSwitchWorkspace}
            disabled={submitting}
          >
            <SelectTrigger size="sm" className="mt-1 w-full">
              <SelectValue>
                {(name) => {
                  const ws = workspaceList.find((w) => w.name === name);
                  return ws?.title || ws?.name || name || "";
                }}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {workspaceList.map((ws) => (
                <SelectItem key={ws.name} value={ws.name}>
                  {ws.title || ws.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <p className="text-sm text-main truncate">
            {currentWorkspace.title || currentWorkspace.name}
          </p>
        )}
      </div>
    </div>
  ) : null;

  const goBack = () => {
    router.back();
  };

  const deny = () => {
    setSubmitting(true);
    const form = document.createElement("form");
    form.method = "POST";
    form.action = AUTHORIZE_URL;
    const fields: [string, string][] = [
      ["client_id", clientId],
      ["redirect_uri", redirectUri],
      ["state", oauthState],
      ["code_challenge", codeChallenge],
      ["code_challenge_method", codeChallengeMethod],
      ["resource", resource],
      ["scope", scope],
      ["action", "deny"],
    ];
    for (const [name, value] of fields) {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = name;
      input.value = value;
      form.appendChild(input);
    }
    document.body.appendChild(form);
    form.submit();
  };

  // Four states share this slot. Early returns rather than a ternary chain,
  // so each is named where it is decided and the consent card reads as the
  // ordinary case it is.
  const consentBody = () => {
    if (loading) {
      return (
        <div className="flex justify-center py-8">
          <Loader2 className="size-6 animate-spin" />
        </div>
      );
    }
    if (error) {
      return (
        <div className="text-center py-4">
          <p className="text-error mb-4">{error}</p>
          <Button appearance="outline" onClick={goBack}>
            {t("common.go-back")}
          </Button>
        </div>
      );
    }
    if (mcpInfo?.capability === MCPSetting_Capability.DISABLED) {
      return (
        <MCPConsentDisabled
          workspaceTitle={
            currentWorkspace?.title || currentWorkspace?.name || ""
          }
          workspaceCard={workspaceCard}
          // Not goBack: history leaves the client waiting on a callback that
          // never comes. A deny POST returns access_denied to the registered
          // redirect_uri, which is the answer the OAuth client is blocked on.
          onDismiss={deny}
          dismissing={submitting}
        />
      );
    }
    return (
      <div className="flex flex-col gap-6">
        <div className="text-center">
          <h1 className="text-xl font-semibold text-main mb-2">
            {t("oauth2.consent.title")}
          </h1>
          <p className="text-control">
            {t("oauth2.consent.description", { clientName })}
          </p>
        </div>
        {workspaceCard}
        {/* Only the two ceilings this bundle can describe. A value it
              cannot name — a newer release's, against a cached page — would
              otherwise fall through to the read-only wording and understate
              what was approved. It gets the same card as no policy data,
              which BOT-106 covers. */}
        {mcpInfo &&
        (mcpInfo.capability === MCPSetting_Capability.READ_ONLY ||
          mcpInfo.capability === MCPSetting_Capability.READ_WRITE) ? (
          <MCPConsentCeiling info={mcpInfo} />
        ) : (
          <div className="bg-control-bg rounded-sm p-4">
            <p className="text-sm text-control-light mb-2">
              {t("oauth2.consent.permissions")}
            </p>
            <ul className="text-sm text-main flex flex-col gap-1">
              <li className="flex items-center gap-2">
                <Check className="size-4 text-success" />
                {t("oauth2.consent.permission-access")}
              </li>
            </ul>
          </div>
        )}
        <form method="POST" action={AUTHORIZE_URL}>
          <input type="hidden" name="client_id" value={clientId} />
          <input type="hidden" name="redirect_uri" value={redirectUri} />
          <input type="hidden" name="state" value={oauthState} />
          <input type="hidden" name="code_challenge" value={codeChallenge} />
          <input
            type="hidden"
            name="code_challenge_method"
            value={codeChallengeMethod}
          />
          <input type="hidden" name="resource" value={resource} />
          <input type="hidden" name="scope" value={scope} />
          <div className="flex gap-x-2">
            <Button
              type="button"
              appearance="outline"
              size="lg"
              className="flex-1"
              disabled={submitting}
              onClick={deny}
            >
              {t("common.deny")}
            </Button>
            <Button
              type="submit"
              size="lg"
              className="flex-1"
              disabled={submitting}
              name="action"
              value="allow"
            >
              {t("common.allow")}
            </Button>
          </div>
        </form>
      </div>
    );
  };

  return (
    // SplashLayout's root is overflow-hidden, so this column carries its own
    // scroll: the ceiling panel and its caution make the card taller than a
    // short viewport, and the part that clips is Allow and Deny. The auto
    // margins keep it centred while it still fits.
    <div className="h-full overflow-y-auto flex flex-col mx-auto w-full max-w-sm py-8">
      <BytebaseLogo className="mx-auto mb-8 mt-auto shrink-0" />
      <div className="rounded-sm border border-control-border bg-white p-6 mb-auto shrink-0">
        {consentBody()}
      </div>
    </div>
  );
}
