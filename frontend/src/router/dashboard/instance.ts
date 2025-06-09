import type { RouteRecordRaw } from "vue-router";
import InstanceLayout from "@/layouts/InstanceLayout.vue";
import { t } from "@/plugins/i18n";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import { INSTANCE_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const INSTANCE_ROUTE_DETAIL = `${INSTANCE_ROUTE_DASHBOARD}.detail`;

const instanceRoutes: RouteRecordRaw[] = [
  {
    path: "instances/:instanceId",
    components: {
      content: InstanceLayout,
      leftSidebar: DashboardSidebar,
    },
    meta: {
      title: () => t("common.instance"),
    },
    props: { content: true },
    children: [
      {
        path: "",
        name: INSTANCE_ROUTE_DETAIL,
        meta: {
          requiredPermissionList: () => ["bb.instances.get"],
        },
        component: () => import("@/views/InstanceDetail.vue"),
        props: true,
      },
    ],
  },
];

export default instanceRoutes;
