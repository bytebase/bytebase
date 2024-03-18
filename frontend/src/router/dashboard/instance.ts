import { RouteRecordRaw } from "vue-router";
import InstanceLayout from "@/layouts/InstanceLayout.vue";
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
    props: { content: true },
    children: [
      {
        path: "",
        name: INSTANCE_ROUTE_DETAIL,
        meta: {
          requiredWorkspacePermissionList: () => ["bb.instances.get"],
        },
        component: () => import("@/views/InstanceDetail.vue"),
        props: true,
      },
    ],
  },
];

export default instanceRoutes;
