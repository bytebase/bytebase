import { mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { defineComponent, h, nextTick } from "vue";
import { createMemoryHistory, createRouter, RouterView } from "vue-router";
import { useBodyLayoutContext } from "./common";

const mocks = vi.hoisted(() => ({
  width: { value: 1024 },
  pushNotification: vi.fn(),
  tryToRemindRelease: vi.fn(async () => false),
  tryToRemindRefresh: vi.fn(async () => false),
}));

vi.mock("@vueuse/core", () => ({
  useWindowSize: () => ({
    width: mocks.width,
  }),
}));

vi.mock("@/react/ReactPageMount.vue", async () => {
  const { defineComponent, h, onMounted, ref } = await import("vue");

  return {
    default: defineComponent({
      name: "MockReactPageMount",
      props: {
        page: {
          type: String,
          required: true,
        },
        pageProps: {
          type: Object,
          default: undefined,
        },
      },
      setup(props) {
        const desktopSidebar = ref<HTMLDivElement | null>(null);
        const mobileSidebar = ref<HTMLDivElement | null>(null);
        const content = ref<HTMLDivElement | null>(null);
        const quickstart = ref<HTMLDivElement | null>(null);
        const mainContainer = ref<HTMLDivElement | null>(null);
        const permissionTarget = ref<HTMLDivElement | null>(null);

        onMounted(() => {
          if (props.page === "DashboardFrameShell") {
            return;
          }
          if (props.page === "Quickstart") {
            return;
          }
          if (props.page === "RoutePermissionGuardShell") {
            props.pageProps?.onReady?.(permissionTarget.value);
            return;
          }

          props.pageProps?.onReady?.({
            desktopSidebar:
              props.pageProps.variant === "workspace"
                ? desktopSidebar.value
                : null,
            mobileSidebar:
              props.pageProps.variant === "workspace"
                ? mobileSidebar.value
                : null,
            content: content.value,
            quickstart: quickstart.value,
            mainContainer: mainContainer.value,
          });
        });

        if (props.page === "Quickstart") {
          // Match the marker the legacy `<Quickstart />` Vue mock emitted
          // — this lets the existing teleport assertions keep working
          // unchanged after the React port replaced the Vue component
          // with `<ReactPageMount page="Quickstart">` in BodyLayout.vue
          // and ReactRouteShellBridge.vue.
          return () => h("div", { "data-testid": "quickstart" }, "quickstart");
        }
        return () =>
          props.page === "RoutePermissionGuardShell"
            ? h("div", {
                ref: permissionTarget,
                "data-testid": "permission-guard",
                class: props.pageProps?.targetClassName,
              })
            : h("div", { "data-testid": "mock-shell" }, [
                h("div", {
                  ref: desktopSidebar,
                  "data-testid": "shell-desktop-sidebar",
                }),
                h("div", {
                  ref: mobileSidebar,
                  "data-testid": "shell-mobile-sidebar",
                }),
                h(
                  "div",
                  {
                    id: "bb-layout-main",
                    ref: mainContainer,
                  },
                  [
                    h("div", {
                      ref: content,
                      "data-testid": "shell-content",
                    }),
                  ]
                ),
                h("div", {
                  ref: quickstart,
                  "data-testid": "shell-quickstart",
                }),
              ]);
      },
    }),
  };
});

vi.mock("@/plugins/i18n", () => ({
  t: (key: string) => key,
}));

vi.mock("@/router/dashboard/workspaceRoutes", () => ({
  WORKSPACE_ROOT_MODULE: "workspace.root",
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
  useActuatorV1Store: () => ({
    tryToRemindRelease: mocks.tryToRemindRelease,
    tryToRemindRefresh: mocks.tryToRemindRefresh,
  }),
  usePermissionStore: () => ({
    currentRolesInWorkspace: new Set<string>(),
    currentPermissions: new Set<string>(["bb.test"]),
    currentPermissionsInProjectV1: () => new Set<string>(["bb.test"]),
  }),
  useSubscriptionV1Store: () => ({
    currentPlan: 0,
  }),
}));

vi.mock("@/types", () => ({
  PresetRoleType: {
    WORKSPACE_ADMIN: "WORKSPACE_ADMIN",
  },
  BASIC_WORKSPACE_PERMISSIONS: [],
}));

vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanType: {
    ENTERPRISE: 1,
  },
}));

import ReactRouteShellBridge from "@/react/ReactRouteShellBridge.vue";
import BodyLayout from "./BodyLayout.vue";

const SidebarView = defineComponent({
  name: "SidebarView",
  setup() {
    return () => h("div", { "data-testid": "sidebar-view" }, "sidebar");
  },
});

const ContentView = defineComponent({
  name: "ContentView",
  setup() {
    const context = useBodyLayoutContext();
    return () =>
      h("div", { "data-testid": "content-view" }, [
        "content:",
        context.mainContainerRef.value ===
        document.getElementById("bb-layout-main")
          ? "connected"
          : "missing",
      ]);
  },
});

async function mountRouteTree(
  routes: Parameters<typeof createRouter>[0]["routes"],
  target = "/target"
) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes,
  });
  await router.push(target);
  await router.isReady();

  const wrapper = mount(RouterView, {
    attachTo: document.body,
    global: {
      plugins: [router],
    },
  });
  await nextTick();
  return wrapper;
}

beforeEach(() => {
  mocks.width.value = 1024;
  mocks.pushNotification.mockClear();
  mocks.tryToRemindRefresh.mockClear();
  mocks.tryToRemindRelease.mockClear();
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("layout bridges", () => {
  test("BodyLayout teleports sidebar, content, and quickstart into the shell", async () => {
    const wrapper = await mountRouteTree([
      {
        path: "/",
        component: BodyLayout,
        children: [
          {
            path: "target",
            name: "workspace.projects",
            components: {
              leftSidebar: SidebarView,
              content: ContentView,
            },
          },
        ],
      },
    ]);

    expect(
      document.querySelector(
        '[data-testid="shell-desktop-sidebar"] [data-testid="sidebar-view"]'
      )
    ).not.toBeNull();
    expect(
      document.querySelector(
        '[data-testid="shell-content"] [data-testid="permission-guard"] [data-testid="content-view"]'
      )?.textContent
    ).toContain("connected");
    expect(
      document.querySelector(
        '[data-testid="shell-quickstart"] [data-testid="quickstart"]'
      )
    ).not.toBeNull();

    wrapper.unmount();
  });

  test("BodyLayout lets project routes reach their project permission shell", async () => {
    const wrapper = await mountRouteTree(
      [
        {
          path: "/",
          component: BodyLayout,
          children: [
            {
              path: "target/:projectId",
              name: "workspace.projects.detail",
              components: {
                leftSidebar: SidebarView,
                content: ContentView,
              },
            },
          ],
        },
      ],
      "/target/prod"
    );

    expect(
      document.querySelector(
        '[data-testid="shell-content"] > [data-testid="content-view"]'
      )?.textContent
    ).toContain("connected");
    expect(
      document.querySelector(
        '[data-testid="shell-content"] [data-testid="permission-guard"]'
      )
    ).toBeNull();

    wrapper.unmount();
  });

  test("ReactRouteShellBridge teleports named route content into React shell", async () => {
    const wrapper = await mountRouteTree([
      {
        path: "/",
        component: ReactRouteShellBridge,
        props: {
          page: "IssuesRouteShell",
          routerViewName: "content",
        },
        children: [
          {
            path: "target",
            name: "issues.target",
            components: {
              leftSidebar: SidebarView,
              content: ContentView,
            },
          },
        ],
      },
    ]);

    expect(
      document.querySelector(
        '[data-testid="shell-desktop-sidebar"] [data-testid="sidebar-view"]'
      )
    ).toBeNull();
    expect(
      document.querySelector(
        '[data-testid="shell-content"] [data-testid="content-view"]'
      )?.textContent
    ).toContain("connected");
    expect(
      document.querySelector(
        '[data-testid="shell-quickstart"] [data-testid="quickstart"]'
      )
    ).not.toBeNull();

    wrapper.unmount();
  });
});
