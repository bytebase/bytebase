import { flatten } from "lodash-es";
import {
  GalleryHorizontalEndIcon,
  SettingsIcon,
  CircleDotIcon,
} from "lucide-vue-next";
import { computed, h, type VNode } from "vue";
import { t } from "@/plugins/i18n";
import {
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_USERS,
  WORKSPACE_ROUTE_SQL_REVIEW,
} from "@/router/dashboard/workspaceRoutes";
import { useCurrentUserV1 } from "@/store";
import {
  useDynamicLocalStorage,
  useDashboardSidebar,
  type DashboardSidebarItem,
} from "@/utils";

export interface QuickLink {
  id: string;
  title: string;
  route?: string;
  icon: () => VNode;
}

export function useQuickLink() {
  const currentUser = useCurrentUserV1();
  const { dashboardSidebarItemList } = useDashboardSidebar();

  const getAccessListBySidebar = (
    sidebarItem: DashboardSidebarItem,
    icon: () => VNode = () => h(SettingsIcon)
  ): QuickLink[] => {
    if (sidebarItem.type === "route" && sidebarItem.title && sidebarItem.name) {
      if (sidebarItem.name === WORKSPACE_ROUTE_LANDING) {
        return [];
      }
      return [
        {
          id: sidebarItem.name,
          title: sidebarItem.title,
          route: sidebarItem.name,
          icon: sidebarItem.icon || icon,
        },
      ];
    }

    if (sidebarItem.children) {
      return flatten(
        sidebarItem.children.map((child) =>
          getAccessListBySidebar(child, sidebarItem.icon)
        )
      ).filter((item) => item) as QuickLink[];
    }

    return [];
  };

  const fullQuickLinkList = computed((): QuickLink[] => {
    const accessList = flatten(
      dashboardSidebarItemList.value.map((item) => getAccessListBySidebar(item))
    ).filter((item) => item) as QuickLink[];

    accessList.unshift(
      {
        id: "visit-projects",
        title: t("landing.quick-link.visit-prjects"),
        icon: () => h(GalleryHorizontalEndIcon),
      },
      {
        id: "visit-issues",
        title: t("landing.quick-link.visit-issues"),
        icon: () => h(CircleDotIcon),
      }
    );

    return accessList;
  });

  const quickAccessConfig = useDynamicLocalStorage<string[]>(
    computed(() => `bb.quick-access.${currentUser.value.name}`),
    [
      "visit-issues",
      "visit-projects",
      WORKSPACE_ROUTE_USERS,
      WORKSPACE_ROUTE_SQL_REVIEW,
    ]
  );

  const quickLinkList = computed({
    get() {
      return quickAccessConfig.value
        .map((id) => {
          return fullQuickLinkList.value.find((item) => item.id === id);
        })
        .filter((item) => item) as QuickLink[];
    },
    set(list) {
      quickAccessConfig.value = list.map((item) => item.id);
    },
  });

  return {
    quickLinkList,
    quickAccessConfig,
    fullQuickLinkList,
  };
}
