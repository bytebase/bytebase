// Augment vue-router RouteMeta for transitive imports from @/router.
// Simplified version of src/shims-vue-router.d.ts that avoids
// referencing RouteLocationNormalized (not available in plain tsc).
import "vue-router";
declare module "vue-router" {
  interface RouteMeta {
    // biome-ignore lint/suspicious/noExplicitAny: simplified shim for React tsc
    title?: (...args: any[]) => string; // eslint-disable-line @typescript-eslint/no-explicit-any
    requiredPermissionList?: () => string[];
    overrideDocumentTitle?: boolean;
  }
}
