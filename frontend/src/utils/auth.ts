import {
  AUTH_MFA_MODULE,
  AUTH_OAUTH_CALLBACK_MODULE,
  AUTH_OIDC_CALLBACK_MODULE,
  AUTH_PASSWORD_FORGOT_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_SIGNIN_ADMIN_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
} from "@/router/auth";
import { router } from "@/router";
import { useActuatorV1Store } from "@/store";

export const isAuthRelatedRoute = (routeName: string) => {
  return [
    AUTH_SIGNIN_MODULE,
    AUTH_SIGNIN_ADMIN_MODULE,
    AUTH_SIGNUP_MODULE,
    AUTH_MFA_MODULE,
    AUTH_PASSWORD_RESET_MODULE,
    AUTH_PASSWORD_FORGOT_MODULE,
    AUTH_OAUTH_CALLBACK_MODULE,
    AUTH_OIDC_CALLBACK_MODULE,
  ].includes(routeName);
};

export const resolveWorkspaceName = (): string | undefined => {
  const actuatorStore = useActuatorV1Store();
  const route = router.currentRoute.value;

  const workspaceID = route.query["workspace"] as string | undefined;
  const workspaceNameFromQuery = workspaceID
    ? `workspaces/${workspaceID}`
    : undefined;
  return actuatorStore.workspaceResourceName || workspaceNameFromQuery;
};