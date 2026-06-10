import { create } from "@bufbuild/protobuf";
import { Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { resolveWorkspaceName } from "@/react/lib/workspace";
import { router } from "@/react/router";
import { AUTH_SIGNIN_MODULE } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import type { OAuthState, OAuthWindowEventPayload } from "@/types/oauth";
import { LoginRequestSchema } from "@/types/proto-es/v1/auth_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";
import { clearOAuthState, retrieveOAuthState } from "@/utils/sso";

type ProcessedOutcome = {
  hasError: boolean;
  messageKey: string;
  oAuthState: OAuthState | undefined;
};

// The OAuth `state` token is single-use: processing consumes it from
// localStorage (`clearOAuthState`). The page component, however, can mount
// more than once per callback navigation — StrictMode double-invokes effects,
// and the app shell can remount routed content while the kicked-off `login()`
// is still running. Re-processing would find the token already consumed and
// misreport a fatal-looking "session expired" error mid-login. Memoizing the
// outcome at module scope makes processing once-per-page-load (the standard
// redirect-callback contract): later mounts replay the outcome render-only,
// leaving the side effects (opener dispatch / login) to the run that computed
// it.
const processedOutcomes = new Map<string, ProcessedOutcome>();

export function OAuthCallbackPage() {
  const { t } = useTranslation();
  const [messageKey, setMessageKey] = useState("");
  const [hasError, setHasError] = useState(false);
  const [oAuthState, setOAuthState] = useState<OAuthState | undefined>(
    undefined
  );
  const [showCloseButton, setShowCloseButton] = useState(false);
  const payloadRef = useRef<OAuthWindowEventPayload>({ error: "", code: "" });

  useEffect(() => {
    const query = router.currentRoute.value.query;
    const stateToken = typeof query.state === "string" ? query.state : "";

    const replayed = processedOutcomes.get(stateToken);
    if (replayed) {
      setHasError(replayed.hasError);
      setMessageKey(replayed.messageKey);
      setOAuthState(replayed.oAuthState);
      return;
    }

    const apply = (outcome: ProcessedOutcome) => {
      processedOutcomes.set(stateToken, outcome);
      setHasError(outcome.hasError);
      setMessageKey(outcome.messageKey);
      setOAuthState(outcome.oAuthState);
    };

    if (stateToken.length === 0) {
      apply({
        hasError: true,
        messageKey: "auth.oauth-callback.invalid-state",
        oAuthState: undefined,
      });
      triggerAuthCallback(undefined, true);
      return;
    }

    const storedState = retrieveOAuthState(stateToken);
    if (!storedState) {
      apply({
        hasError: true,
        messageKey: "auth.oauth-callback.session-expired",
        oAuthState: undefined,
      });
      triggerAuthCallback(undefined, true);
      return;
    }

    if (storedState.token !== stateToken) {
      apply({
        hasError: true,
        messageKey: "auth.oauth-callback.security-failed",
        oAuthState: undefined,
      });
      clearOAuthState(stateToken);
      triggerAuthCallback(undefined, true);
      return;
    }

    apply({
      hasError: false,
      messageKey: "auth.oauth-callback.success-redirecting",
      oAuthState: storedState,
    });
    payloadRef.current.code = (query.code as string) || "";
    clearOAuthState(storedState.token);
    triggerAuthCallback(storedState, false);
  }, []);

  const triggerAuthCallback = async (
    state: OAuthState | undefined,
    isError: boolean
  ) => {
    if (state?.popup) {
      try {
        if (!window.opener || window.opener.closed) {
          setHasError(true);
          setMessageKey("auth.oauth-callback.opener-unavailable");
          setShowCloseButton(true);
          return;
        }

        const eventName = isError ? "bb.oauth.unknown" : state.event;

        window.opener.dispatchEvent(
          new CustomEvent(eventName, { detail: payloadRef.current })
        );

        try {
          window.close();

          setTimeout(() => {
            if (!window.closed) {
              setShowCloseButton(true);
              if (!isError) {
                setMessageKey("auth.oauth-callback.success-close");
              }
            }
          }, 500);
        } catch {
          setShowCloseButton(true);
          if (!isError) {
            setMessageKey("auth.oauth-callback.please-close");
          }
        }
      } catch (error) {
        console.error("Failed to communicate with opener window:", error);
        setHasError(true);
        setMessageKey("auth.oauth-callback.opener-failed");
        setShowCloseButton(true);
      }
      return;
    }

    if (isError || !state) {
      return;
    }

    const eventName = state.event;
    if (eventName.startsWith("bb.oauth.signin")) {
      const isOidc = state.idpType === IdentityProviderType.OIDC;
      const idpName = eventName.split(".").pop();
      if (!idpName) {
        return;
      }

      await useAppStore.getState().login({
        request: create(LoginRequestSchema, {
          idpName,
          idpContext: {
            context: {
              case: isOidc ? "oidcContext" : "oauth2Context",
              value: {
                code: payloadRef.current.code,
              },
            },
          },
          workspace: resolveWorkspaceName(),
        }),
        redirect: true,
        redirectUrl: state.redirect,
      });
    }
  };

  const backToSignin = () => {
    router.push({ name: AUTH_SIGNIN_MODULE });
  };

  return (
    <div className="p-4">
      {hasError ? (
        <div className="mt-2">
          <div>{messageKey && t(messageKey)}</div>
          {oAuthState?.popup ? (
            <Button onClick={() => window.close()}>{t("common.close")}</Button>
          ) : (
            <button type="button" className="btn-normal" onClick={backToSignin}>
              {t("auth.back-to-signin")}
            </button>
          )}
        </div>
      ) : (
        <div className="mt-2">
          <div className="flex items-center gap-x-2">
            <Loader2 className="size-4 animate-spin" />
            <span>{t(messageKey || "auth.oauth-callback.processing")}</span>
          </div>
          {oAuthState?.popup && showCloseButton && (
            <Button className="mt-4" onClick={() => window.close()}>
              {t("auth.close-window")}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
