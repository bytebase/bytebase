import type { RouteRecordRaw } from "vue-router";
import SplashLayout from "@/layouts/SplashLayout.vue";
import { t } from "@/plugins/i18n";

export const AUTH_SIGNIN_MODULE = "auth.signin";
export const AUTH_SIGNIN_ADMIN_MODULE = "auth.signin.admin";
export const AUTH_SIGNUP_MODULE = "auth.signup";
export const AUTH_MFA_MODULE = "auth.mfa";
export const AUTH_PASSWORD_RESET_MODULE = "auth.password.reset";
export const AUTH_PASSWORD_FORGOT_MODULE = "auth.password.forgot";
export const AUTH_OAUTH_CALLBACK_MODULE = "auth.oauth.callback";
export const AUTH_OIDC_CALLBACK_MODULE = "auth.oidc.callback";
export const AUTH_IDP_INIT_MODULE = "auth.idp.init";
export const AUTH_2FA_SETUP_MODULE = "auth.2fa.setup";
export const OAUTH2_CONSENT_MODULE = "oauth2.consent";

const authRoutes: RouteRecordRaw[] = [
  {
    path: "/oauth2/consent",
    component: SplashLayout,
    children: [
      {
        path: "",
        name: OAUTH2_CONSENT_MODULE,
        meta: { title: () => t("oauth2.consent.title") },
        component: () => import("@/views/OAuth2Consent.vue"),
      },
    ],
  },
  {
    path: "/auth",
    name: "auth",
    component: SplashLayout,
    children: [
      {
        path: "",
        alias: "signin",
        name: AUTH_SIGNIN_MODULE,
        meta: { title: () => t("common.sign-in") },
        component: () => import("@/views/auth/Signin.vue"),
      },
      {
        // We need the admin as the backdoor for the workspace admin.
        path: "admin",
        name: AUTH_SIGNIN_ADMIN_MODULE,
        meta: { title: () => t("common.sign-in-as-admin") },
        component: () => import("@/views/auth/SigninAdmin.vue"),
      },
      {
        path: "signup",
        name: AUTH_SIGNUP_MODULE,
        meta: { title: () => t("common.sign-up") },
        component: () => import("@/views/auth/Signup.vue"),
      },
      {
        path: "password-forgot",
        name: AUTH_PASSWORD_FORGOT_MODULE,
        meta: { title: () => `${t("auth.password-forgot")}` },
        component: () => import("@/views/auth/PasswordForgot.vue"),
      },
      {
        path: "password-reset",
        name: AUTH_PASSWORD_RESET_MODULE,
        meta: { title: () => `${t("auth.password-reset.title")}` },
        component: () => import("@/views/auth/PasswordReset.vue"),
      },
      {
        path: "mfa",
        name: AUTH_MFA_MODULE,
        meta: { title: () => t("multi-factor.self") },
        component: () => import("@/views/auth/MultiFactor.vue"),
      },
      {
        path: "idp-init",
        name: AUTH_IDP_INIT_MODULE,
        meta: { title: () => "Initializing SSO" },
        component: () => import("@/views/IdPInitiatedSSO.vue"),
      },
    ],
  },
  {
    path: "/oauth/callback",
    name: AUTH_OAUTH_CALLBACK_MODULE,
    component: () => import("@/views/OAuthCallback.vue"),
  },
  {
    path: "/oidc/callback",
    name: AUTH_OIDC_CALLBACK_MODULE,
    component: () => import("@/views/OAuthCallback.vue"),
  },
  {
    path: "/2fa/setup",
    name: AUTH_2FA_SETUP_MODULE,
    meta: { title: () => t("two-factor.self") },
    component: () => import("@/views/TwoFactorRequired.vue"),
  },
];

export default authRoutes;
