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
import DatabaseLayout from "../layouts/DatabaseLayout.vue";
import InstanceLayout from "../layouts/InstanceLayout.vue";
import DashboardSidebar from "../views/DashboardSidebar.vue";
import Home from "../views/Home.vue";
import Signin from "../views/auth/Signin.vue";
import Signup from "../views/auth/Signup.vue";
import Activate from "../views/auth/Activate.vue";
import PasswordReset from "../views/auth/PasswordReset.vue";
import { store } from "../store";
import { isDev, idFromSlug, isDBAOrOwner } from "../utils";
import { Principal } from "../types";

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
    component: DashboardLayout,
    children: [
      {
        path: "",
        components: { body: BodyLayout },
        children: [
          {
            path: "",
            name: HOME_MODULE,
            meta: {
              quickActionListByRole: () => {
                return new Map([
                  [
                    "OWNER",
                    [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.create",
                      "quickaction.bb.database.troubleshoot",
                      "quickaction.bb.instance.create",
                      "quickaction.bb.project.create",
                      "quickaction.bb.user.manage",
                    ],
                  ],
                  [
                    "DBA",
                    [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.create",
                      "quickaction.bb.database.troubleshoot",
                      "quickaction.bb.instance.create",
                      "quickaction.bb.project.create",
                    ],
                  ],
                  [
                    "DEVELOPER",
                    [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.request",
                      "quickaction.bb.database.troubleshoot",
                      "quickaction.bb.project.create",
                    ],
                  ],
                ]);
              },
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
            path: "403",
            name: "error.403",
            components: {
              content: () => import("../views/Page403.vue"),
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
            path: "archive",
            name: "workspace.archive",
            meta: { title: () => "Archive" },
            components: {
              content: () => import("../views/Archive.vue"),
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
                path: "version-control",
                name: "setting.workspace.version-control",
                meta: { title: () => "Version Control" },
                component: () => import("../views/SettingWorkspaceVCS.vue"),
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
              quickActionListByRole: () => {
                return new Map([
                  [
                    "OWNER",
                    [
                      "quickaction.bb.environment.create",
                      "quickaction.bb.environment.reorder",
                    ],
                  ],
                  [
                    "DBA",
                    [
                      "quickaction.bb.environment.create",
                      "quickaction.bb.environment.reorder",
                    ],
                  ],
                ]);
              },
            },
            components: {
              content: () => import("../views/EnvironmentDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "environment/:environmentSlug",
            name: "workspace.environment.detail",
            meta: {
              title: (route: RouteLocationNormalized) => {
                const slug = route.params.environmentSlug as string;
                return store.getters["environment/environmentById"](
                  idFromSlug(slug)
                ).name;
              },
              allowBookmark: true,
            },
            components: {
              content: () => import("../views/EnvironmentDetail.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
          },
          {
            path: "project",
            name: "workspace.project",
            meta: {
              title: () => "Project",
              quickActionListByRole: () => {
                return new Map([
                  ["OWNER", ["quickaction.bb.project.create"]],
                  ["DBA", ["quickaction.bb.project.create"]],
                  ["DEVELOPER", ["quickaction.bb.project.create"]],
                ]);
              },
            },
            components: {
              content: () => import("../views/ProjectDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "project/:projectSlug",
            name: "workspace.project.detail",
            meta: {
              title: (route: RouteLocationNormalized) => {
                const slug = route.params.projectSlug as string;
                return store.getters["project/projectById"](idFromSlug(slug))
                  .name;
              },
              quickActionListByRole: (route: RouteLocationNormalized) => {
                const slug = route.params.projectSlug as string;
                const project = store.getters["project/projectById"](
                  idFromSlug(slug)
                );

                if (project.rowStatus == "NORMAL") {
                  return new Map([
                    ["OWNER", ["quickaction.bb.database.schema.update"]],
                    ["DBA", ["quickaction.bb.database.schema.update"]],
                    ["DEVELOPER", ["quickaction.bb.database.schema.update"]],
                  ]);
                }
                return new Map();
              },
              allowBookmark: true,
            },
            components: {
              content: () => import("../views/ProjectDetail.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
          },
          {
            path: "instance",
            name: "workspace.instance",
            meta: {
              title: () => "Instance",
              quickActionListByRole: () => {
                return new Map([
                  ["OWNER", ["quickaction.bb.instance.create"]],
                  ["DBA", ["quickaction.bb.instance.create"]],
                ]);
              },
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
              quickActionListByRole: () => {
                return new Map([
                  [
                    "OWNER",
                    [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.create",
                    ],
                  ],
                  [
                    "DBA",
                    [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.create",
                    ],
                  ],
                  [
                    "DEVELOPER",
                    [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.request",
                      "quickaction.bb.database.troubleshoot",
                    ],
                  ],
                ]);
              },
            },
            components: {
              content: () => import("../views/DatabaseDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "db/new",
            name: "workspace.database.create",
            meta: {
              title: () => "Create database",
            },
            components: {
              content: () => import("../views/DatabaseCreate.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "db/grant",
            name: "workspace.database.grant",
            meta: {
              title: () => "Grant database",
            },
            components: {
              content: () => import("../views/DatabaseGrant.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "db/:databaseSlug",
            name: "workspace.database.detail",
            components: {
              content: DatabaseLayout,
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
            children: [
              {
                path: "",
                name: "workspace.database.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.databaseSlug as string;
                    if (slug.toLowerCase() == "new") {
                      return "New";
                    }
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
                path: "datasource/:dataSourceSlug",
                name: "workspace.database.datasource.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.dataSourceSlug as string;
                    if (slug.toLowerCase() == "new") {
                      return "New";
                    }
                    return (
                      "Data source - " +
                      store.getters["dataSource/dataSourceById"](
                        idFromSlug(slug)
                      ).name
                    );
                  },
                  allowBookmark: true,
                },
                component: () => import("../views/DataSourceDetail.vue"),
                props: true,
              },
            ],
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
            ],
          },
          {
            path: "issue/:issueSlug",
            name: "workspace.issue.detail",
            meta: {
              title: (route: RouteLocationNormalized) => {
                const slug = route.params.issueSlug as string;
                if (slug.toLowerCase() == "new") {
                  return "New";
                }
                return store.getters["issue/issueById"](idFromSlug(slug)).name;
              },
              allowBookmark: true,
            },
            components: {
              content: () => import("../views/IssueDetail.vue"),
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
  const isLoggedIn = store.getters["auth/isLoggedIn"]();

  const fromModule = from.name
    ? from.name.toString().split(".")[0]
    : HOME_MODULE;
  const toModule = to.name ? to.name.toString().split(".")[0] : HOME_MODULE;

  if (toModule != fromModule) {
    store.dispatch("router/setBackPath", from.fullPath);
  }

  if (
    to.name === SIGNIN_MODULE ||
    to.name === SIGNUP_MODULE ||
    to.name === ACTIVATE_MODULE ||
    to.name === PASSWORD_RESET_MODULE
  ) {
    if (isLoggedIn) {
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
    if (!isLoggedIn) {
      next({ name: SIGNIN_MODULE, replace: true });
      return;
    }
  }

  const loginUser: Principal = store.getters["auth/currentUser"]();
  if (to.name === "workspace.instance") {
    if (isDBAOrOwner(loginUser.role)) {
      next();
    } else {
      next({
        name: "error.403",
        replace: false,
      });
    }
    return;
  }

  // Fetching specific instance would try to fetch the sensitive info
  // so we short circuit here for non-DBA/Owner to avoid there.
  if (to.name?.toString().startsWith("workspace.instance.")) {
    if (!isDBAOrOwner(loginUser.role)) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name === "workspace.database.create") {
    if (
      !store.getters["plan/feature"]("bb.dba-workflow") ||
      isDBAOrOwner(loginUser.role)
    ) {
      next();
    } else {
      next({
        name: "error.403",
        replace: false,
      });
    }
    return;
  }

  if (to.name?.toString().startsWith("workspace.database.datasource")) {
    if (
      !store.getters["plan/feature"]("bb.data-source") ||
      !isDBAOrOwner(loginUser.role)
    ) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (
    to.name === "error.403" ||
    to.name === "error.404" ||
    to.name === "error.500" ||
    to.name === "workspace.home" ||
    to.name === "workspace.inbox" ||
    to.name === "workspace.project" ||
    to.name === "workspace.database" ||
    to.name === "workspace.archive" ||
    to.name === "workspace.environment" ||
    to.name?.toString().startsWith("setting")
  ) {
    next();
    return;
  }

  // We may just change the anchor (e.g. in Issue Detail view), thus we don't need
  // to fetch the data to verify its existence since we have already verified before.
  if (to.name == from.name) {
    next();
    return;
  }

  const routerSlug = store.getters["router/routeSlug"](to);
  const environmentSlug = routerSlug.environmentSlug;
  const projectSlug = routerSlug.projectSlug;
  const issueSlug = routerSlug.issueSlug;
  const instanceSlug = routerSlug.instanceSlug;
  const databaseSlug = routerSlug.databaseSlug;
  const dataSourceSlug = routerSlug.dataSourceSlug;
  const principalId = routerSlug.principalId;

  console.debug("RouterSlug:", routerSlug);

  if (environmentSlug) {
    if (
      store.getters["environment/environmentById"](idFromSlug(environmentSlug))
    ) {
      next();
      return;
    }
    next({
      name: "error.404",
      replace: false,
    });
  }

  if (projectSlug) {
    store
      .dispatch("project/fetchProjectById", idFromSlug(projectSlug))
      .then((project) => {
        next();
      })
      .catch((error) => {
        next({
          name: "error.404",
          replace: false,
        });
        throw error;
      });
    return;
  }

  if (issueSlug) {
    if (issueSlug.toLowerCase() == "new") {
      next();
      return;
    }
    store
      .dispatch("issue/fetchIssueById", idFromSlug(issueSlug))
      .then((issue) => {
        next();
      })
      .catch((error) => {
        next({
          name: "error.404",
          replace: false,
        });
        throw error;
      });
    return;
  }

  if (databaseSlug) {
    if (
      databaseSlug.toLowerCase() == "new" ||
      databaseSlug.toLowerCase() == "grant"
    ) {
      next();
      return;
    }
    store
      .dispatch("database/fetchDatabaseById", {
        databaseId: idFromSlug(databaseSlug),
      })
      .then((database) => {
        if (!dataSourceSlug) {
          next();
        } else {
          store
            .dispatch("dataSource/fetchDataSourceById", {
              dataSourceId: idFromSlug(dataSourceSlug),
              databaseId: database.id,
            })
            .then((dataSource) => {
              next();
            })
            .catch((error) => {
              next({
                name: "error.404",
                replace: false,
              });
              throw error;
            });
        }
      })
      .catch((error) => {
        next({
          name: "error.404",
          replace: false,
        });
        throw error;
      });
    return;
  }

  if (instanceSlug) {
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
        throw error;
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
        throw error;
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
