import { RouteRecordRaw } from "vue-router";
import SplashLayout from "@/layouts/SplashLayout.vue";
import { t } from "@/plugins/i18n";

export const AUTH_SIGNIN_MODULE = "auth.signin";
export const AUTH_SIGNUP_MODULE = "auth.signup";
export const AUTH_MFA_MODULE = "auth.mfa";
export const AUTH_PASSWORD_FORGOT_MODULE = "auth.password.forgot";
export const AUTH_OAUTH_CALLBACK_MODULE = "auth.oauth.callback";
export const AUTH_OIDC_CALLBACK_MODULE = "auth.oidc.callback";
export const AUTH_2FA_SETUP_MODULE = "auth.2fa.setup";

const authRoutes: RouteRecordRaw[] = [
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
        path: "mfa",
        name: AUTH_MFA_MODULE,
        meta: { title: () => t("multi-factor.self") },
        component: () => import("@/views/auth/MultiFactor.vue"),
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
