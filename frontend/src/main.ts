// Must be first - initializes global compatibility shims
import "./init";
import "regenerator-runtime/runtime";
import "./assets/css/github-markdown-style.css";
import "./assets/css/tailwind.css";
// Side-effect: configures the shared dayjs singleton (localizedFormat) and
// registers the SQL highlight.js language + theme used across the app.
import "./plugins/dayjs";
import "./plugins/highlight";
// Side-effect: registers the bb.vue-notification window listener and
// constructs the toastManager singleton. Must load before any auth/error
// interceptor can fire pushNotification during bootstrap RPCs.
import "./react/lib/toast";
import { setActivePinia } from "pinia";
import { useAppStore } from "./react/stores/app";
import { pinia } from "./store";
import { isDev, isRelease, migrateStorageKeys } from "./utils";

console.debug("dev:", isDev());
console.debug("release:", isRelease());

// Transition shim: activate Pinia outside a Vue app so the Pinia stores still
// referenced by shared `.ts` utils keep working during the Vue→React
// migration. Removed in teardown once no Pinia store remains.
setActivePinia(pinia);

// Migrate renamed localStorage keys before any store reads from storage.
migrateStorageKeys();

(async () => {
  const store = useAppStore.getState();

  // Load the authenticated session BEFORE mounting so the route guard sees the
  // correct auth state on the very first navigation (the Vue bootstrap awaited
  // fetchCurrentUser + server info the same way before mounting the router).
  const currentUser = await store.loadCurrentUser();
  const initPromises: Promise<unknown>[] = [store.loadServerInfo()];
  if (currentUser) {
    initPromises.push(store.loadSubscription());
    initPromises.push(store.loadWorkspaceList());
    initPromises.push(store.loadWorkspaceProfile());
  }
  await Promise.all(initPromises);

  const { mountReactRouterApp } = await import("./react/app/mountApp");
  await mountReactRouterApp("#app");
})();
