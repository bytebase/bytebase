import type { VNode } from "vue";

export interface SidebarItem {
  title?: string;
  // If the type is route, we'd like to use the name instead of the path
  name?: string;
  // path is required if the type is div or link
  path?: string;
  icon?: () => VNode;
  hide?: boolean;
  type: "route" | "div" | "divider" | "link";
  expand?: boolean;
  children?: SidebarItem[];
}
