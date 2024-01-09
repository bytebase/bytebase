import { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import { useEnvironmentV1Store } from "@/store";
import { QuickActionType } from "@/types";
import { idFromSlug } from "@/utils";
import DashboardSidebar from "@/views/DashboardSidebar.vue";

const environmentRoutes: RouteRecordRaw[] = [
  {
    path: "environment",
    name: "workspace.environment",
    meta: {
      title: () => t("common.environments"),
      quickActionListByRole: () => {
        const DBA_AND_OWNER_QUICK_ACTION_LIST: QuickActionType[] = [
          "quickaction.bb.environment.create",
          "quickaction.bb.environment.reorder",
        ];
        return new Map([
          ["OWNER", DBA_AND_OWNER_QUICK_ACTION_LIST],
          ["DBA", DBA_AND_OWNER_QUICK_ACTION_LIST],
          ["DEVELOPER", []],
        ]);
      },
    },
    components: {
      content: () => import("@/views/EnvironmentDashboard.vue"),
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
        return useEnvironmentV1Store().getEnvironmentByUID(
          String(idFromSlug(slug))
        ).title;
      },
    },
    components: {
      content: () => import("@/views/EnvironmentDetail.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true },
  },
];

export default environmentRoutes;
