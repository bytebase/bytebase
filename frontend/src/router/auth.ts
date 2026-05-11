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
export const AUTH_PROFILE_SETUP_MODULE = "auth.profile.setup";
export const AUTH_2FA_SETUP_MODULE = "auth.2fa.setup";
export const OAUTH2_CONSENT_MODULE = "oauth2.consent";

// Every auth/consent route renders a React page mounted by ReactPageMount.vue.
// The `page` prop names the React component registered in @/react/mount.ts.
const reactPage = () => import("@/react/ReactPageMount.vue");

const authRoutes: RouteRecordRaw[] = [
  {
    path: "/oauth2/consent",
    component: SplashLayout,
    children: [
      {
        path: "",
        name: OAUTH2_CONSENT_MODULE,
        meta: { title: () => t("oauth2.consent.title") },
        component: reactPage,
        props: { page: "OAuth2ConsentPage" },
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
        component: reactPage,
        props: { page: "SigninPage" },
      },
      {
        // We need the admin as the backdoor for the workspace admin.
        path: "admin",
        name: AUTH_SIGNIN_ADMIN_MODULE,
        meta: { title: () => t("common.sign-in-as-admin") },
        component: reactPage,
        props: { page: "SigninAdminPage" },
      },
      {
        path: "signup",
        name: AUTH_SIGNUP_MODULE,
        meta: { title: () => t("common.sign-up") },
        component: reactPage,
        props: { page: "SignupPage" },
      },
      {
        path: "password-forgot",
        name: AUTH_PASSWORD_FORGOT_MODULE,
        meta: { title: () => `${t("auth.password-forgot")}` },
        component: reactPage,
        props: { page: "PasswordForgotPage" },
      },
      {
        path: "password-reset",
        name: AUTH_PASSWORD_RESET_MODULE,
        meta: { title: () => `${t("auth.password-reset.title")}` },
        component: reactPage,
        props: { page: "PasswordResetPage" },
      },
      {
        path: "mfa",
        name: AUTH_MFA_MODULE,
        meta: { title: () => t("multi-factor.self") },
        component: reactPage,
        props: { page: "MultiFactorPage" },
      },
      {
        path: "profile-setup",
        name: AUTH_PROFILE_SETUP_MODULE,
        meta: { title: () => t("auth.profile-setup") },
        component: reactPage,
        props: { page: "ProfileSetupPage" },
      },
    ],
  },
  {
    path: "/oauth/callback",
    name: AUTH_OAUTH_CALLBACK_MODULE,
    component: reactPage,
    props: { page: "OAuthCallbackPage" },
  },
  {
    path: "/oidc/callback",
    name: AUTH_OIDC_CALLBACK_MODULE,
    component: reactPage,
    props: { page: "OAuthCallbackPage" },
  },
  {
    path: "/2fa/setup",
    name: AUTH_2FA_SETUP_MODULE,
    meta: { title: () => t("two-factor.self") },
    component: reactPage,
    props: { page: "TwoFactorRequiredPage" },
  },
];

export default authRoutes;
