import type { RouteObject } from "react-router";
import { SplashLayout } from "@/app/layouts/SplashLayout";
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
} from "@/app/router/handles";
import { lazyPage } from "@/app/router/lazyPage";

// Authentication, consent, and initial workspace setup routes.
export const authRoutes: RouteObject[] = [
  {
    path: "/oauth2/consent",
    element: <SplashLayout />,
    children: [
      {
        index: true,
        handle: { name: OAUTH2_CONSENT_MODULE },
        lazy: lazyPage(
          () => import("@/routes/auth/OAuth2ConsentPage"),
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
          () => import("@/routes/auth/SigninPage"),
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
          () => import("@/routes/auth/SigninPage"),
          (m) => m.SigninPage
        ),
      },
      {
        path: "admin",
        handle: { name: AUTH_SIGNIN_ADMIN_MODULE },
        lazy: lazyPage(
          () => import("@/routes/auth/SigninAdminPage"),
          (m) => m.SigninAdminPage
        ),
      },
      {
        path: "signup",
        handle: { name: AUTH_SIGNUP_MODULE },
        lazy: lazyPage(
          () => import("@/routes/auth/SignupPage"),
          (m) => m.SignupPage
        ),
      },
      {
        path: "password-forgot",
        handle: { name: AUTH_PASSWORD_FORGOT_MODULE },
        lazy: lazyPage(
          () => import("@/routes/auth/PasswordForgotPage"),
          (m) => m.PasswordForgotPage
        ),
      },
      {
        path: "password-reset",
        handle: { name: AUTH_PASSWORD_RESET_MODULE },
        lazy: lazyPage(
          () => import("@/routes/auth/PasswordResetPage"),
          (m) => m.PasswordResetPage
        ),
      },
      {
        path: "mfa",
        handle: { name: AUTH_MFA_MODULE },
        lazy: lazyPage(
          () => import("@/routes/auth/MultiFactorPage"),
          (m) => m.MultiFactorPage
        ),
      },
      {
        path: "profile-setup",
        handle: { name: AUTH_PROFILE_SETUP_MODULE },
        lazy: lazyPage(
          () => import("@/routes/auth/ProfileSetupPage"),
          (m) => m.ProfileSetupPage
        ),
      },
    ],
  },
  {
    path: "/oauth/callback",
    handle: { name: AUTH_OAUTH_CALLBACK_MODULE },
    lazy: lazyPage(
      () => import("@/routes/auth/OAuthCallbackPage"),
      (m) => m.OAuthCallbackPage
    ),
  },
  {
    path: "/oidc/callback",
    handle: { name: AUTH_OIDC_CALLBACK_MODULE },
    lazy: lazyPage(
      () => import("@/routes/auth/OAuthCallbackPage"),
      (m) => m.OAuthCallbackPage
    ),
  },
  {
    path: "/2fa/setup",
    handle: { name: AUTH_2FA_SETUP_MODULE },
    lazy: lazyPage(
      () => import("@/routes/auth/TwoFactorRequiredPage"),
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
          () => import("@/routes/auth/SetupPage"),
          (m) => m.SetupPage
        ),
      },
    ],
  },
];
