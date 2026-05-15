import { Building2, Check, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { BytebaseLogo } from "@/react/components/BytebaseLogo";
import { Button } from "@/react/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useActuatorV1Store, useAuthStore, useWorkspaceV1Store } from "@/store";

const AUTHORIZE_URL = "/api/oauth2/authorize";

export function OAuth2ConsentPage() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [clientName, setClientName] = useState("");

  // Pinia store singletons are resolved once at the top of the component so
  // the useVueState selector closures don't repeat the use*-prefixed call —
  // which trips the React-hooks-in-callback lint rule (typescript:S6440)
  // despite Pinia memoizing the factory.
  const authStore = useAuthStore();
  const actuatorStore = useActuatorV1Store();
  const workspaceStore = useWorkspaceV1Store();

  const isLoggedIn = useVueState(() => authStore.isLoggedIn);
  // Workspace context shown on the consent card. On SaaS, every Bytebase
  // user belongs to at least one workspace; on self-hosted there's a single
  // implicit workspace. We display it so the user can confirm which
  // workspace this OAuth grant will be bound to.
  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
  const currentWorkspace = useVueState(() => workspaceStore.currentWorkspace);
  const workspaceList = useVueState(() => workspaceStore.workspaceList);

  const query = router.currentRoute.value.query;
  const clientId = (query.client_id as string) || "";
  const redirectUri = (query.redirect_uri as string) || "";
  const oauthState = (query.state as string) || "";
  const codeChallenge = (query.code_challenge as string) || "";
  const codeChallengeMethod = (query.code_challenge_method as string) || "";

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
        setError(t("oauth2.consent.error-load-failed"));
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
    workspaceStore.fetchWorkspaceList().catch(() => {});
  }, [isSaaSMode, workspaceStore]);

  // Switch the active workspace in-place, preserving the consent flow.
  // Uses the workspace store's withoutRedirect variant so that
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
      await workspaceStore.switchWorkspaceWithoutRedirect(workspaceName);
      globalThis.location.reload();
    } catch {
      setError(t("oauth2.consent.error-switch-failed"));
      setSubmitting(false);
    }
  };

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

  return (
    <div className="h-full flex flex-col justify-center mx-auto w-full max-w-sm">
      <BytebaseLogo className="mx-auto mb-8" />
      <div className="rounded-sm border border-control-border bg-white p-6">
        {loading ? (
          <div className="flex justify-center py-8">
            <Loader2 className="size-6 animate-spin" />
          </div>
        ) : error ? (
          <div className="text-center py-4">
            <p className="text-error mb-4">{error}</p>
            <Button variant="outline" onClick={goBack}>
              {t("common.go-back")}
            </Button>
          </div>
        ) : (
          <div className="space-y-6">
            <div className="text-center">
              <h1 className="text-xl font-semibold text-main mb-2">
                {t("oauth2.consent.title")}
              </h1>
              <p className="text-control">
                {t("oauth2.consent.description", { clientName })}
              </p>
            </div>
            {currentWorkspace && (
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
                            const ws = workspaceList.find(
                              (w) => w.name === name
                            );
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
            )}
            <div className="bg-control-bg rounded-sm p-4">
              <p className="text-sm text-control-light mb-2">
                {t("oauth2.consent.permissions")}
              </p>
              <ul className="text-sm text-main space-y-1">
                <li className="flex items-center gap-2">
                  <Check className="size-4 text-success" />
                  {t("oauth2.consent.permission-access")}
                </li>
              </ul>
            </div>
            <form method="POST" action={AUTHORIZE_URL}>
              <input type="hidden" name="client_id" value={clientId} />
              <input type="hidden" name="redirect_uri" value={redirectUri} />
              <input type="hidden" name="state" value={oauthState} />
              <input
                type="hidden"
                name="code_challenge"
                value={codeChallenge}
              />
              <input
                type="hidden"
                name="code_challenge_method"
                value={codeChallengeMethod}
              />
              <div className="flex gap-x-2">
                <Button
                  type="button"
                  variant="outline"
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
        )}
      </div>
    </div>
  );
}
