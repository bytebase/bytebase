import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  PROJECT_V1_ROUTE_WEBHOOK_CREATE,
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
  PROJECT_V1_ROUTE_WEBHOOKS,
} from "./handles";
import { router } from "./index";
import { setAppRouter, setRouteNameIndex } from "./navigation";

beforeEach(() => {
  setRouteNameIndex(
    new Map<string, string>([
      [PROJECT_V1_ROUTE_WEBHOOKS, "/projects/:projectId/webhooks"],
      [PROJECT_V1_ROUTE_WEBHOOK_CREATE, "/projects/:projectId/webhooks/new"],
      [
        PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
        "/projects/:projectId/webhooks/:webhookResourceId",
      ],
    ])
  );
  setAppRouter({
    navigate: vi.fn(),
    state: {
      location: {
        pathname: "/projects/project-sample/webhooks",
        search: "",
        hash: "",
      },
      matches: [
        {
          handle: { name: PROJECT_V1_ROUTE_WEBHOOKS },
          params: { projectId: "project-sample" },
        },
      ],
      initialized: true,
    },
  });
});

// Guardrail for the "getSnapshot should be cached" infinite-loop class.
// `router.currentRoute.value` backs the `useReactiveRoute` `useSyncExternalStore`
// snapshot. If it returns a fresh object on every read, any external-store
// consumer re-renders forever (this crashed the SQL Editor: GutterBar →
// useReactiveRoute → "Maximum update depth exceeded"). The snapshot is memoized in
// `currentRouteSnapshot` to keep a stable reference between actual route
// changes; this test fails if that memoization regresses.
describe("router.currentRoute snapshot stability", () => {
  it("returns a referentially stable object across reads without navigation", () => {
    const a = router.currentRoute.value;
    const b = router.currentRoute.value;
    expect(a).toBe(b);
  });
});

describe("router named route params", () => {
  it("inherits current params when resolving project child routes", () => {
    expect(
      router.resolve({ name: PROJECT_V1_ROUTE_WEBHOOK_CREATE }).fullPath
    ).toBe("/projects/project-sample/webhooks/new");
    expect(
      router.resolve({
        name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
        params: { webhookResourceId: "hook-1" },
      }).fullPath
    ).toBe("/projects/project-sample/webhooks/hook-1");
  });
});
