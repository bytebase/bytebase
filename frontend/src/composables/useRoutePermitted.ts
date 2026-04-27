import {
  type ComputedRef,
  computed,
  type MaybeRefOrGetter,
  toValue,
} from "vue";
import { useRoute } from "vue-router";
import { usePermissionStore } from "@/store";
import { BASIC_WORKSPACE_PERMISSIONS, type Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

// Mirrors useComponentPermissionState in the React shell. Runs in Vue
// reactive context so the parent layout can gate `<router-view>` mount
// synchronously on route change — otherwise an unauthorized route's
// component can mount and fire setup-time requests in the window between
// the route change and the React shell's async onReady(null).
export function useRoutePermitted(
  project?: MaybeRefOrGetter<Project | undefined>
): ComputedRef<boolean> {
  const route = useRoute();
  const permissionStore = usePermissionStore();

  return computed(() => {
    const workspacePermissions = permissionStore.currentPermissions;
    if (
      !BASIC_WORKSPACE_PERMISSIONS.every((p) => workspacePermissions.has(p))
    ) {
      return false;
    }

    const required = route.matched.flatMap(
      (record) => (record.meta.requiredPermissionList?.() ?? []) as Permission[]
    );
    if (required.length === 0) {
      return true;
    }

    const resolvedProject = project ? toValue(project) : undefined;
    const projectPermissions = resolvedProject
      ? permissionStore.currentPermissionsInProjectV1(resolvedProject)
      : null;
    return required.every(
      (p) => workspacePermissions.has(p) || !!projectPermissions?.has(p)
    );
  });
}
