import { defineAsyncComponent } from "vue";
import { createRouter, createWebHistory, RouteRecordRaw } from "vue-router";
import MainContent from "../views/MainContent.vue";
import MainSidebar from "../views/MainSidebar.vue";
import EmptyView from "../views/EmptyView.vue";
import ActivitySidebar from "../views/ActivitySidebar.vue";
import Home from "../views/Home.vue";
import ActionbarHome from "../views/ActionbarHome.vue";
import Signin from "../views/auth/Signin.vue";
import Signup from "../views/auth/Signup.vue";
import PasswordReset from "../views/auth/PasswordReset.vue";
import MasterDetail from "../views/MasterDetail.vue";
import { store } from "../store";
import { Group, Project } from "../types";

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
    component: MasterDetail,
    children: [
      {
        path: "404",
        name: "error.404",
        meta: { displayName: "404" },
        components: {
          content: defineAsyncComponent(() => import("../views/Page404.vue")),
          leftsidebar: MainSidebar,
          rightsidebar: ActivitySidebar,
        },
      },
      {
        path: "500",
        name: "error.500",
        meta: { displayName: "500" },
        components: {
          content: defineAsyncComponent(() => import("../views/Page500.vue")),
          leftsidebar: MainSidebar,
          rightsidebar: ActivitySidebar,
        },
      },
      {
        path: "inbox",
        name: "workspace.inbox",
        meta: { displayName: "Inbox" },
        components: {
          content: defineAsyncComponent(() => import("../views/Inbox.vue")),
          leftsidebar: MainSidebar,
          rightsidebar: ActivitySidebar,
        },
        props: { actionbar: true, content: true },
      },
      {
        path: "environment",
        name: "workspace.environment",
        meta: { displayName: "Environment" },
        components: {
          content: defineAsyncComponent(
            () => import("../views/EnvironmentDashboard.vue")
          ),
          leftsidebar: MainSidebar,
          rightsidebar: ActivitySidebar,
        },
        props: { actionbar: true, content: true },
      },
      {
        path: "setting",
        name: "setting",
        meta: { displayName: "Setting" },
        components: {
          content: defineAsyncComponent(() => import("../views/Setting.vue")),
          leftsidebar: defineAsyncComponent(
            () => import("../views/SettingSidebar.vue")
          ),
          rightsidebar: ActivitySidebar,
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
          {
            // Use absolute path to specify the whole path while still leverages the nested route
            path: "/:groupSlug/:projectSlug/setting",
            name: "setting.workspace.project",
            meta: { displayName: "Setting" },
            component: defineAsyncComponent(
              () => import("../views/SettingWorkspaceProject.vue")
            ),
            props: true,
          },
        ],
      },
      {
        path: "",
        name: HOME_MODULE,
        components: {
          content: MainContent,
          leftsidebar: MainSidebar,
          rightsidebar: ActivitySidebar,
        },

        children: [
          {
            path: "",
            name: HOME_MODULE,
            meta: { displayName: "Home" },
            components: {
              actionbar: ActionbarHome,
              content: Home,
            },
            props: { actionbar: true, content: true },
          },
          {
            path: "pipeline/:pipelineId",
            name: "workspace.pipeline.detail",
            meta: { displayName: "Pipeline" },
            components: {
              actionbar: EmptyView,
              content: defineAsyncComponent(
                () => import("../views/PipelineDetail.vue")
              ),
            },
            props: { actionbar: true, content: true },
          },
          {
            path: ":groupSlug",
            name: "workspace.group",
            meta: { displayName: "Dashboard" },
            components: {
              actionbar: defineAsyncComponent(
                () => import("../views/ActionbarGroup.vue")
              ),
              content: defineAsyncComponent(
                () => import("../views/GroupDashboard.vue")
              ),
            },
            props: { actionbar: true, content: true },
          },
          {
            path: ":groupSlug/setting",
            name: "workspace.group.setting",
            meta: { displayName: "Setting" },
            components: {
              actionbar: EmptyView,
              content: defineAsyncComponent(
                () => import("../views/GroupSetting.vue")
              ),
            },
            props: { actionbar: true, content: true },
          },
          {
            path: ":groupSlug/:projectSlug/repository",
            name: "workspace.project.repository",
            meta: { displayName: "Repository" },
            components: {
              actionbar: EmptyView,
              content: defineAsyncComponent(
                () => import("../views/RepositoryDashboard.vue")
              ),
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
  linkActiveClass: "text-gray-900 bg-link-hover",
  linkExactActiveClass: "text-gray-900 bg-link-hover",
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
  const groupSlug = routerSlug.groupSlug;
  const projectSlug = routerSlug.projectSlug;

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

  store
    .dispatch("group/fetchGroupListForUser", loginUser.id)
    .then((list: Group[]) => {
      let matchedGroup;
      if (groupSlug) {
        matchedGroup = list.find(
          (group: Group) => group.attributes.slug === groupSlug
        );
      }

      if (!matchedGroup) {
        next({
          name: "error.404",
          replace: false,
        });
        return;
      }

      // If no project path is specified, just go to group default page
      if (!projectSlug || to.name === "workspace.group.setting") {
        next();
        return;
      }

      let matchedProject;
      store
        .dispatch("project/fetchProjectListForGroup", matchedGroup.id)
        .then((list: Project[]) => {
          matchedProject = list.find(
            (project: Project) => project.attributes.slug === projectSlug
          );

          if (!matchedProject) {
            next({
              name: "error.404",
              replace: false,
            });
            return;
          }

          next();
          return;
        })
        .catch((error: Error) => {
          console.log("error", typeof error);
          next(error);
          return;
        });
    })
    .catch((error: Error) => {
      next(error);
      return;
    });
});
