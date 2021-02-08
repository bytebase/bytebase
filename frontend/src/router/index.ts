import { defineAsyncComponent, nextTick } from "vue";
import {
  createRouter,
  createWebHistory,
  RouteLocationNormalized,
  RouteRecordRaw,
} from "vue-router";
import DashboardLayout from "../layouts/DashboardLayout.vue";
import BodyLayout from "../layouts/BodyLayout.vue";
import DashboardSidebar from "../views/DashboardSidebar.vue";
import Home from "../views/Home.vue";
import Signin from "../views/auth/Signin.vue";
import Signup from "../views/auth/Signup.vue";
import PasswordReset from "../views/auth/PasswordReset.vue";
import { store } from "../store";
import { idFromSlug } from "../utils";

const HOME_MODULE = "workspace.home";
const SIGNIN_MODULE = "auth.signin";
const SIGNUP_MODULE = "auth.signup";
const PASSWORD_RESET_MODULE = "auth.password.reset";

const routes: Array<RouteRecordRaw> = [
  {
    path: "/signin",
    name: SIGNIN_MODULE,
    meta: { title: () => "Signin" },
    component: Signin,
  },
  {
    path: "/signup",
    name: SIGNUP_MODULE,
    meta: { title: () => "Signup" },
    component: Signup,
  },
  {
    path: "/password-reset",
    name: PASSWORD_RESET_MODULE,
    meta: { title: () => "Reset Password" },
    component: PasswordReset,
  },
  {
    path: "/",
    name: HOME_MODULE,
    component: DashboardLayout,
    children: [
      {
        path: "",
        name: HOME_MODULE,
        components: { body: BodyLayout },
        children: [
          {
            path: "",
            name: HOME_MODULE,
            meta: {
              quickActionList: [
                "instance.create",
                "user.manage",
                "datasource.create",
                "datasource.schema.edit",
                "ticket.create",
              ],
            },
            components: {
              content: Home,
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "404",
            name: "error.404",
            components: {
              content: defineAsyncComponent(
                () => import("../views/Page404.vue")
              ),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "500",
            name: "error.500",
            components: {
              content: defineAsyncComponent(
                () => import("../views/Page500.vue")
              ),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "inbox",
            name: "workspace.inbox",
            meta: { title: () => "Inbox" },
            components: {
              content: defineAsyncComponent(() => import("../views/Inbox.vue")),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "setting",
            name: "setting",
            meta: { title: () => "Setting" },
            components: {
              content: defineAsyncComponent(
                () => import("../layouts/SettingLayout.vue")
              ),
              leftSidebar: defineAsyncComponent(
                () => import("../views/SettingSidebar.vue")
              ),
              props: {
                content: true,
                leftSidebar: true,
              },
            },
            children: [
              {
                path: "",
                name: "setting.accountprofile",
                meta: { title: () => "Account Profile" },
                component: defineAsyncComponent(
                  () => import("../views/SettingAccountProfile.vue")
                ),
                alias: "account/profile",
                props: true,
              },
              {
                path: "general",
                name: "setting.workspace.general",
                meta: { title: () => "Account General" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspaceGeneral.vue")
                ),
                props: true,
              },
              {
                path: "agent",
                name: "setting.workspace.agent",
                meta: { title: () => "Agents" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspaceAgent.vue")
                ),
                props: true,
              },
              {
                path: "member",
                name: "setting.workspace.member",
                meta: { title: () => "Members" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspaceMember.vue")
                ),
                props: true,
              },
              {
                path: "plan",
                name: "setting.workspace.plan",
                meta: { title: () => "Plans" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspacePlan.vue")
                ),
                props: true,
              },
              {
                path: "billing",
                name: "setting.workspace.billing",
                meta: { title: () => "Billings" },
                component: defineAsyncComponent(
                  () => import("../views/SettingWorkspaceBilling.vue")
                ),
                props: true,
              },
              {
                path: "integration/slack",
                name: "setting.workspace.integration.slack",
                meta: { title: () => "Slack" },
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
            meta: {
              title: () => "Environment",
              quickActionList: ["environment.create", "environment.reorder"],
            },
            components: {
              content: defineAsyncComponent(
                () => import("../views/EnvironmentDashboard.vue")
              ),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "instance",
            name: "workspace.instance",
            meta: {
              title: () => "Instance",
              quickActionList: ["instance.create"],
            },
            components: {
              content: defineAsyncComponent(
                () => import("../views/InstanceDashboard.vue")
              ),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "instance/:instanceSlug",
            name: "workspace.instance.detail",
            meta: {
              title: (route: RouteLocationNormalized) => {
                const slug = route.params.instanceSlug as string;
                if (slug.toUpperCase() == "NEW") {
                  return "New";
                }
                return store.getters["instance/instanceById"](idFromSlug(slug))
                  .attributes.name;
              },
            },
            components: {
              content: defineAsyncComponent(
                () => import("../views/InstanceDetail.vue")
              ),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
          },
          {
            path: "task/:taskId",
            name: "workspace.task.detail",
            meta: {
              title: (route: RouteLocationNormalized) => {
                const taskId = route.params.taskId as string;
                if (taskId.toUpperCase() == "NEW") {
                  return "New";
                }
                return store.getters["task/taskById"](taskId).attributes.name;
              },
            },
            components: {
              content: defineAsyncComponent(
                () => import("../views/TaskDetail.vue")
              ),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
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
    to.name === "workspace.instance" ||
    to.name?.toString().startsWith("setting")
  ) {
    next();
    return;
  }

  const routerSlug = store.getters["router/routeSlug"](to);
  const taskId = routerSlug.taskId;
  const instanceSlug = routerSlug.instanceSlug;

  console.log("RouterSlug:", routerSlug);

  if (taskId) {
    if (taskId.toUpperCase() == "NEW") {
      next();
      return;
    }
    store
      .dispatch("task/fetchTaskById", taskId)
      .then((task) => {
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

  if (instanceSlug) {
    if (instanceSlug.toUpperCase() == "NEW") {
      next();
      return;
    }
    store
      .dispatch("instance/fetchInstanceById", idFromSlug(instanceSlug))
      .then((instance) => {
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

  next({
    name: "error.404",
    replace: false,
  });
});

router.afterEach((to, from) => {
  // Needs to use nextTick otherwise title will still be the one from the previous route.
  nextTick(() => {
    if (to.meta.title) {
      document.title = to.meta.title(to);
    } else {
      document.title = "Bytebase";
    }
  });
});
