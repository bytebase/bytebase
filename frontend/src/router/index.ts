import { defineAsyncComponent } from "vue";
import { createRouter, createWebHistory, RouteRecordRaw } from "vue-router";
import DashboardLayout from "../layouts/DashboardLayout.vue";
import ThreeColumnView from "../views/ThreeColumnView.vue";
import MainSidebar from "../views/MainSidebar.vue";
import ActivitySidebar from "../views/ActivitySidebar.vue";
import Home from "../views/Home.vue";
import Signin from "../views/auth/Signin.vue";
import Signup from "../views/auth/Signup.vue";
import PasswordReset from "../views/auth/PasswordReset.vue";
import { store } from "../store";

const HOME_MODULE = "workspace.home";
const SIGNIN_MODULE = "auth.signin";
const SIGNUP_MODULE = "auth.signup";
const PASSWORD_RESET_MODULE = "auth.password.reset";

const routes: Array<RouteRecordRaw> = [
  {
    path: "/signin",
    name: SIGNIN_MODULE,
    meta: { displayName: "Signin" },
    component: Signin,
  },
  {
    path: "/signup",
    name: SIGNUP_MODULE,
    meta: { displayName: "Signup" },
    component: Signup,
  },
  {
    path: "/password-reset",
    name: PASSWORD_RESET_MODULE,
    meta: { displayName: "Reset Password" },
    component: PasswordReset,
  },
  {
    path: "/",
    name: HOME_MODULE,
    component: DashboardLayout,
    children: [
      {
        path: "404",
        name: "error.404",
        meta: { displayName: "404" },
        component: defineAsyncComponent(() => import("../views/Page404.vue")),
      },
      {
        path: "500",
        name: "error.500",
        meta: { displayName: "500" },
        component: defineAsyncComponent(() => import("../views/Page500.vue")),
      },
      {
        path: "inbox",
        name: "workspace.inbox",
        meta: { displayName: "Inbox" },
        component: defineAsyncComponent(() => import("../views/Inbox.vue")),
        props: { actionbar: true, content: true },
      },
      {
        path: "",
        name: HOME_MODULE,
        component: ThreeColumnView,

        children: [
          {
            path: "",
            name: HOME_MODULE,
            meta: { displayName: "Home", rightSidebar: true },
            components: {
              content: Home,
              leftSidebar: MainSidebar,
              rightSidebar: ActivitySidebar,
            },
            props: { actionbar: true, content: true },
          },
          {
            path: "setting",
            name: "setting",
            meta: { displayName: "Setting" },
            components: {
              content: defineAsyncComponent(
                () => import("../views/Setting.vue")
              ),
              leftSidebar: defineAsyncComponent(
                () => import("../views/SettingSidebar.vue")
              ),
            },
            children: [
              {
                path: "",
                name: "setting.accountprofile",
                meta: { displayName: "Account Profile" },
                component: defineAsyncComponent(
                  () => import("../views/SettingAccountProfile.vue")
                ),
                alias: "account/profile",
                props: true,
              },
              {
                path: "general",
                name: "setting.workspace.general",
                meta: { displayName: "General" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspaceGeneral.vue")
                ),
                props: true,
              },
              {
                path: "agent",
                name: "setting.workspace.agent",
                meta: { displayName: "Agents" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspaceAgent.vue")
                ),
                props: true,
              },
              {
                path: "member",
                name: "setting.workspace.member",
                meta: { displayName: "Members" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspaceMember.vue")
                ),
                props: true,
              },
              {
                path: "plan",
                name: "setting.workspace.plan",
                meta: { displayName: "Plans" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspacePlan.vue")
                ),
                props: true,
              },
              {
                path: "billing",
                name: "setting.workspace.billing",
                meta: { displayName: "Billings" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspaceBilling.vue")
                ),
                props: true,
              },
              {
                path: "integration/slack",
                name: "setting.workspace.integration.slack",
                meta: { displayName: "Slack" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspaceIntegrationSlack.vue")
                ),
                props: true,
              },
            ],
          },
          {
            path: "environment",
            name: "workspace.environment",
            meta: { displayName: "Environment" },
            components: {
              content: defineAsyncComponent(
                () => import("../views/EnvironmentDashboard.vue")
              ),
              leftSidebar: MainSidebar,
            },
            props: { actionbar: true, content: true },
          },
          {
            path: "pipeline/:pipelineId",
            name: "workspace.pipeline.detail",
            meta: { displayName: "Pipeline" },
            components: {
              content: defineAsyncComponent(
                () => import("../views/PipelineDetail.vue")
              ),
              leftSidebar: MainSidebar,
            },
            props: { actionbar: true, content: true },
          },
        ],
      },
    ],
  },
];

export const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
  linkExactActiveClass: "bg-link-hover",
});

router.beforeEach((to, from, next) => {
  const loginUser = store.getters["auth/currentUser"]();
  console.log("LoginUser:", loginUser);

  const fromModule = from.name
    ? from.name.toString().split(".")[0]
    : HOME_MODULE;
  const toModule = to.name ? to.name.toString().split(".")[0] : HOME_MODULE;

  console.log("Route module:", fromModule, "->", toModule);

  if (toModule != fromModule) {
    store.dispatch("router/setBackPath", from.fullPath);
  }

  if (
    to.name === SIGNIN_MODULE ||
    to.name === SIGNUP_MODULE ||
    to.name === PASSWORD_RESET_MODULE
  ) {
    if (loginUser) {
      next({ name: HOME_MODULE, replace: true });
    } else {
      next();
    }
    return;
  } else {
    if (!loginUser) {
      next({ name: SIGNIN_MODULE, replace: true });
      return;
    }
  }

  if (
    to.name === "error.404" ||
    to.name === "error.500" ||
    to.name === "workspace.home" ||
    to.name === "workspace.inbox" ||
    to.name === "workspace.environment" ||
    to.name?.toString().startsWith("setting")
  ) {
    next();
    return;
  }

  const routerSlug = store.getters["router/routeSlug"](to);
  const pipelineId = routerSlug.pipelineId;

  console.log("RouterSlug:", routerSlug);

  if (pipelineId) {
    store
      .dispatch("pipeline/fetchPipelineById", pipelineId)
      .then((pipeline) => {
        next();
      })
      .catch((error) => {
        next({
          name: "error.404",
          replace: false,
        });
      });
    return;
  }

  next();
});
