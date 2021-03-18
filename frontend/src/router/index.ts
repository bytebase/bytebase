import { nextTick } from "vue";
import {
  createRouter,
  createWebHistory,
  RouteLocationNormalized,
  RouteRecordRaw,
} from "vue-router";
import SplashLayout from "../layouts/SplashLayout.vue";
import DashboardLayout from "../layouts/DashboardLayout.vue";
import BodyLayout from "../layouts/BodyLayout.vue";
import InstanceLayout from "../layouts/InstanceLayout.vue";
import DashboardSidebar from "../views/DashboardSidebar.vue";
import Home from "../views/Home.vue";
import Signin from "../views/auth/Signin.vue";
import Signup from "../views/auth/Signup.vue";
import Activate from "../views/auth/Activate.vue";
import PasswordReset from "../views/auth/PasswordReset.vue";
import { store } from "../store";
import { isDev, idFromSlug } from "../utils";
import { User } from "../types";

const HOME_MODULE = "workspace.home";
const AUTH_MODULE = "auth";
const SIGNIN_MODULE = "auth.signin";
const SIGNUP_MODULE = "auth.signup";
const ACTIVATE_MODULE = "auth.activate";
const PASSWORD_RESET_MODULE = "auth.password.reset";

const routes: Array<RouteRecordRaw> = [
  {
    path: "/auth",
    name: AUTH_MODULE,
    component: SplashLayout,
    children: [
      {
        path: "",
        name: SIGNIN_MODULE,
        meta: { title: () => "Signin" },
        component: Signin,
        alias: "signin",
        props: true,
      },
      {
        path: "signup",
        name: SIGNUP_MODULE,
        meta: { title: () => "Signup" },
        component: Signup,
        props: true,
      },
      {
        path: "activate",
        name: ACTIVATE_MODULE,
        meta: { title: () => "Activate" },
        component: Activate,
        props: true,
      },
      {
        path: "password-reset",
        name: PASSWORD_RESET_MODULE,
        meta: { title: () => "Reset Password" },
        component: PasswordReset,
        props: true,
      },
    ],
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
                "datasource.request",
                "datasource.schema.update",
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
              content: () => import("../views/Page404.vue"),
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
              content: () => import("../views/Page500.vue"),
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
              content: () => import("../views/Inbox.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            // "u" stands for user. Strictly speaking, it's not accurate because we
            // may refer to other principal type in the future. But from the endusers'
            // perspective, they are more familiar with the "user" concept.
            // We make an exception to use a shorthand here because it's a commonly
            // accessed endpint, and maybe in the future, we will further provide a
            // shortlink like u/<<uid>>
            path: "u/:principalId",
            name: "workspace.profile",
            meta: {
              title: (route: RouteLocationNormalized) => {
                const principalId = route.params.principalId as string;
                return store.getters["principal/principalById"](principalId)
                  .name;
              },
            },
            components: {
              content: () => import("../views/ProfileDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
          },
          {
            path: "setting",
            name: "setting",
            meta: { title: () => "Setting" },
            components: {
              content: () => import("../layouts/SettingLayout.vue"),
              leftSidebar: () => import("../views/SettingSidebar.vue"),
            },
            props: {
              content: true,
              leftSidebar: true,
            },
            children: [
              {
                path: "",
                name: "setting.profile",
                meta: { title: () => "Profile" },
                component: () => import("../views/ProfileDashboard.vue"),
                alias: "profile",
                props: true,
              },
              {
                path: "general",
                name: "setting.workspace.general",
                meta: { title: () => "Account General" },
                component: () => import("../views/SettingWorkspaceGeneral.vue"),
                props: true,
              },
              {
                path: "agent",
                name: "setting.workspace.agent",
                meta: { title: () => "Agents" },
                component: () => import("../views/SettingWorkspaceAgent.vue"),
                props: true,
              },
              {
                path: "member",
                name: "setting.workspace.member",
                meta: { title: () => "Members" },
                component: () => import("../views/SettingWorkspaceMember.vue"),
                props: true,
              },
              {
                path: "plan",
                name: "setting.workspace.plan",
                meta: { title: () => "Plans" },
                component: () => import("../views/SettingWorkspacePlan.vue"),
                props: true,
              },
              {
                path: "billing",
                name: "setting.workspace.billing",
                meta: { title: () => "Billings" },
                component: () => import("../views/SettingWorkspaceBilling.vue"),
                props: true,
              },
              {
                path: "integration/slack",
                name: "setting.workspace.integration.slack",
                meta: { title: () => "Slack" },
                component: () =>
                  import("../views/SettingWorkspaceIntegrationSlack.vue"),
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
              content: () => import("../views/EnvironmentDashboard.vue"),
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
              content: () => import("../views/InstanceDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "db",
            name: "workspace.database",
            meta: {
              title: () => "Database",
            },
            components: {
              content: () => import("../views/DatabaseDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "instance/:instanceSlug",
            name: "workspace.instance.detail",
            components: {
              content: InstanceLayout,
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
            children: [
              {
                path: "",
                name: "workspace.instance.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.instanceSlug as string;
                    if (slug.toLowerCase() == "new") {
                      return "New";
                    }
                    return store.getters["instance/instanceById"](
                      idFromSlug(slug)
                    ).name;
                  },
                },
                component: () => import("../views/InstanceDetail.vue"),
                props: true,
              },
              {
                path: "db/:databaseSlug",
                name: "workspace.instance.database.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.databaseSlug as string;
                    return store.getters["database/databaseById"](
                      idFromSlug(slug)
                    ).name;
                  },
                  allowBookmark: true,
                },
                component: () => import("../views/DatabaseDetail.vue"),
                props: true,
              },
              {
                path: "ds/:dataSourceSlug",
                name: "workspace.instance.datasource.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.dataSourceSlug as string;
                    if (slug.toLowerCase() == "new") {
                      return "New";
                    }
                    return store.getters["dataSource/dataSourceById"](
                      idFromSlug(slug)
                    ).name;
                  },
                  allowBookmark: true,
                },
                component: () => import("../views/DataSourceDetail.vue"),
                props: true,
              },
            ],
          },
          {
            path: "task/:taskSlug",
            name: "workspace.task.detail",
            meta: {
              title: (route: RouteLocationNormalized) => {
                const slug = route.params.taskSlug as string;
                if (slug.toLowerCase() == "new") {
                  return "New";
                }
                return store.getters["task/taskById"](idFromSlug(slug)).name;
              },
              allowBookmark: true,
            },
            components: {
              content: () => import("../views/TaskDetail.vue"),
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
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  linkExactActiveClass: "bg-link-hover",
  scrollBehavior(to, from, savedPosition) {
    if (to.hash) {
      return {
        el: to.hash,
        behavior: "smooth",
      };
    }
  },
});

router.beforeEach((to, from, next) => {
  const loginUser: User = store.getters["auth/currentUser"]();
  const isGuest = loginUser.id == "0";

  const fromModule = from.name
    ? from.name.toString().split(".")[0]
    : HOME_MODULE;
  const toModule = to.name ? to.name.toString().split(".")[0] : HOME_MODULE;

  if (isDev()) {
    console.log("LoginUser:", loginUser);
    console.log("Route module:", fromModule, "->", toModule);
  }

  if (toModule != fromModule) {
    store.dispatch("router/setBackPath", from.fullPath);
  }

  if (
    to.name === SIGNIN_MODULE ||
    to.name === SIGNUP_MODULE ||
    to.name === ACTIVATE_MODULE ||
    to.name === PASSWORD_RESET_MODULE
  ) {
    if (!isGuest) {
      next({ name: HOME_MODULE, replace: true });
    } else {
      if (to.name === ACTIVATE_MODULE) {
        const token = to.query.token;
        if (token) {
          // TODO: Needs to validate the token
          next();
        } else {
          // Go to signup if token is missing
          next({ name: SIGNUP_MODULE, replace: true });
        }
      } else {
        next();
      }
    }
    return;
  } else {
    if (isGuest) {
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
    to.name === "workspace.database" ||
    to.name?.toString().startsWith("setting")
  ) {
    next();
    return;
  }

  const routerSlug = store.getters["router/routeSlug"](to);
  const taskSlug = routerSlug.taskSlug;
  const instanceSlug = routerSlug.instanceSlug;
  const databaseSlug = routerSlug.databaseSlug;
  const dataSourceSlug = routerSlug.dataSourceSlug;
  const principalId = routerSlug.principalId;

  console.log("RouterSlug:", routerSlug);

  if (taskSlug) {
    if (taskSlug.toLowerCase() == "new") {
      next();
      return;
    }
    store
      .dispatch("task/fetchTaskById", idFromSlug(taskSlug))
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
    store
      .dispatch("instance/fetchInstanceById", idFromSlug(instanceSlug))
      .then((instance) => {
        if (databaseSlug) {
          store
            .dispatch("database/fetchDatabaseById", {
              instanceId: instance.id,
              databaseId: idFromSlug(databaseSlug),
            })
            .then((database) => {
              next();
            })
            .catch((error) => {
              next({
                name: "error.404",
                replace: false,
              });
            });
        } else if (dataSourceSlug) {
          store
            .dispatch("dataSource/fetchDataSourceById", {
              instanceId: instance.id,
              dataSourceId: idFromSlug(dataSourceSlug),
            })
            .then((dataSource) => {
              next();
            })
            .catch((error) => {
              next({
                name: "error.404",
                replace: false,
              });
            });
        } else {
          next();
        }
      })
      .catch((error) => {
        next({
          name: "error.404",
          replace: false,
        });
      });
    return;
  }

  if (principalId) {
    store
      .dispatch("principal/fetchPrincipalById", principalId)
      .then(() => {
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
