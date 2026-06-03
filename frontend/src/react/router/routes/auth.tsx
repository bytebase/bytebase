import type { RouteObject } from "react-router-dom";
import { SplashLayout } from "@/react/app/layouts/SplashLayout";
import {
  AUTH_2FA_SETUP_MODULE,
  AUTH_MFA_MODULE,
  AUTH_OAUTH_CALLBACK_MODULE,
  AUTH_OIDC_CALLBACK_MODULE,
  AUTH_PASSWORD_FORGOT_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_PROFILE_SETUP_MODULE,
  AUTH_SIGNIN_ADMIN_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
  OAUTH2_CONSENT_MODULE,
  SETUP_MODULE,
} from "@/react/router/handles";
import { lazyPage } from "@/react/router/lazyPage";

// Translated from `@/router/auth.ts` and `@/router/setup.ts`. Every auth /
// consent / setup leaf renders a self-contained React page under the
// SplashLayout chrome.
export const authRoutes: RouteObject[] = [
  {
    path: "/oauth2/consent",
    element: <SplashLayout />,
    children: [
      {
        index: true,
        handle: { name: OAUTH2_CONSENT_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/OAuth2ConsentPage"),
          (m) => m.OAuth2ConsentPage
        ),
      },
    ],
  },
  {
    path: "/auth",
    handle: { name: "auth" },
    element: <SplashLayout />,
    children: [
      {
        index: true,
        handle: { name: AUTH_SIGNIN_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/SigninPage"),
          (m) => m.SigninPage
        ),
      },
      {
        // vue used `alias: "signin"` on the index child; react-router has no
        // alias, so `/auth/signin` is an explicit sibling rendering the same
        // page under the same route name.
        path: "signin",
        handle: { name: AUTH_SIGNIN_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/SigninPage"),
          (m) => m.SigninPage
        ),
      },
      {
        path: "admin",
        handle: { name: AUTH_SIGNIN_ADMIN_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/SigninAdminPage"),
          (m) => m.SigninAdminPage
        ),
      },
      {
        path: "signup",
        handle: { name: AUTH_SIGNUP_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/SignupPage"),
          (m) => m.SignupPage
        ),
      },
      {
        path: "password-forgot",
        handle: { name: AUTH_PASSWORD_FORGOT_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/PasswordForgotPage"),
          (m) => m.PasswordForgotPage
        ),
      },
      {
        path: "password-reset",
        handle: { name: AUTH_PASSWORD_RESET_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/PasswordResetPage"),
          (m) => m.PasswordResetPage
        ),
      },
      {
        path: "mfa",
        handle: { name: AUTH_MFA_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/MultiFactorPage"),
          (m) => m.MultiFactorPage
        ),
      },
      {
        path: "profile-setup",
        handle: { name: AUTH_PROFILE_SETUP_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/ProfileSetupPage"),
          (m) => m.ProfileSetupPage
        ),
      },
    ],
  },
  {
    path: "/oauth/callback",
    handle: { name: AUTH_OAUTH_CALLBACK_MODULE },
    lazy: lazyPage(
      () => import("@/react/pages/auth/OAuthCallbackPage"),
      (m) => m.OAuthCallbackPage
    ),
  },
  {
    path: "/oidc/callback",
    handle: { name: AUTH_OIDC_CALLBACK_MODULE },
    lazy: lazyPage(
      () => import("@/react/pages/auth/OAuthCallbackPage"),
      (m) => m.OAuthCallbackPage
    ),
  },
  {
    path: "/2fa/setup",
    handle: { name: AUTH_2FA_SETUP_MODULE },
    lazy: lazyPage(
      () => import("@/react/pages/auth/TwoFactorRequiredPage"),
      (m) => m.TwoFactorRequiredPage
    ),
  },
  {
    path: "/setup",
    element: <SplashLayout />,
    children: [
      {
        index: true,
        handle: { name: SETUP_MODULE },
        lazy: lazyPage(
          () => import("@/react/pages/auth/SetupPage"),
          (m) => m.SetupPage
        ),
      },
    ],
  },
];
