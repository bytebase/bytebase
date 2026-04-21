import { Check, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { BytebaseLogo } from "@/react/components/BytebaseLogo";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useAuthStore } from "@/store";

const AUTHORIZE_URL = "/api/oauth2/authorize";

export function OAuth2ConsentPage() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [clientName, setClientName] = useState("");

  const isLoggedIn = useVueState(() => useAuthStore().isLoggedIn);

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
      setError("Missing required OAuth2 parameters");
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
          setError(data.error_description || "Client not found");
          setLoading(false);
          return;
        }
        const data = await response.json();
        setClientName(data.client_name || clientId);
      } catch {
        setError("Failed to load client information");
      }
      setLoading(false);
    })();
  }, []);

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
            <div className="bg-control-bg rounded-sm p-4">
              <p className="text-sm text-control-light mb-2">
                {t("oauth2.consent.permissions")}
              </p>
              <ul className="text-sm text-main space-y-1">
                <li className="flex items-center gap-2">
                  <Check className="w-4 h-4 text-success" />
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
