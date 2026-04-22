import { create } from "@bufbuild/protobuf";
import { Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useAuthStore } from "@/store";
import type { OAuthState, OAuthWindowEventPayload } from "@/types/oauth";
import { LoginRequestSchema } from "@/types/proto-es/v1/auth_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";
import { resolveWorkspaceName } from "@/utils";
import { clearOAuthState, retrieveOAuthState } from "@/utils/sso";

export function OAuthCallbackPage() {
  const { t } = useTranslation();
  const [message, setMessage] = useState("");
  const [hasError, setHasError] = useState(false);
  const [oAuthState, setOAuthState] = useState<OAuthState | undefined>(
    undefined
  );
  const [showCloseButton, setShowCloseButton] = useState(false);
  const payloadRef = useRef<OAuthWindowEventPayload>({ error: "", code: "" });
  // StrictMode double-invokes effects; the OAuth state token is single-use
  // (consumed via `clearOAuthState`). Without this guard the second invocation
  // would fail to retrieve the already-cleared token and flash a
  // "session expired" error before the login completes.
  const initRef = useRef(false);

  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;
    const query = router.currentRoute.value.query;
    const stateToken = query.state as string | undefined;

    if (
      !stateToken ||
      typeof stateToken !== "string" ||
      stateToken.length === 0
    ) {
      setHasError(true);
      setMessage(t("auth.oauth-callback.invalid-state"));
      triggerAuthCallback(undefined, true);
      return;
    }

    const storedState = retrieveOAuthState(stateToken);
    if (!storedState) {
      setHasError(true);
      setMessage(t("auth.oauth-callback.session-expired"));
      triggerAuthCallback(undefined, true);
      return;
    }

    if (storedState.token !== stateToken) {
      setHasError(true);
      setMessage(t("auth.oauth-callback.security-failed"));
      clearOAuthState(stateToken);
      triggerAuthCallback(undefined, true);
      return;
    }

    setOAuthState(storedState);
    setHasError(false);
    setMessage(t("auth.oauth-callback.success-redirecting"));
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
          setMessage(t("auth.oauth-callback.opener-unavailable"));
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
                setMessage(t("auth.oauth-callback.success-close"));
              }
            }
          }, 500);
        } catch {
          setShowCloseButton(true);
          if (!isError) {
            setMessage(t("auth.oauth-callback.please-close"));
          }
        }
      } catch (error) {
        console.error("Failed to communicate with opener window:", error);
        setHasError(true);
        setMessage(t("auth.oauth-callback.opener-failed"));
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

      await useAuthStore().login({
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
          <div>{message}</div>
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
            <span>{message || t("auth.oauth-callback.processing")}</span>
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
