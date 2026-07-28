import { createMemoryRouter } from "react-router";
import { afterEach, describe, expect, test } from "vitest";
import { canonicalRedirect } from "./canonicalRedirect";
import { setAppRouter } from "./navigation";

const createRouter = (initialEntries: string[], initialIndex: number) => {
  const router = createMemoryRouter(
    [
      { path: "/before" },
      {
        path: "/legacy",
        loader: () => canonicalRedirect("/canonical"),
      },
      { path: "/canonical" },
      { path: "/after" },
    ],
    { initialEntries, initialIndex }
  );
  setAppRouter(router);
  return router;
};

let router: ReturnType<typeof createMemoryRouter> | undefined;

afterEach(() => {
  router?.dispose();
  router = undefined;
});

describe("canonicalRedirect", () => {
  test("replaces a legacy entry reached by POP so Back can continue", async () => {
    router = createRouter(["/before", "/legacy", "/after"], 2);

    await router.navigate(-1);

    expect(router.state.location.pathname).toBe("/canonical");
    expect(router.state.historyAction).toBe("REPLACE");

    await router.navigate(-1);

    expect(router.state.location.pathname).toBe("/before");
  });

  test("pushes an in-app redirect destination and preserves its referrer", async () => {
    router = createRouter(["/before"], 0);

    await router.navigate("/legacy");

    expect(router.state.location.pathname).toBe("/canonical");
    expect(router.state.historyAction).toBe("PUSH");

    await router.navigate(-1);

    expect(router.state.location.pathname).toBe("/before");
  });

  test("keeps an in-app REPLACE navigation as a replacement", async () => {
    router = createRouter(["/before"], 0);

    await router.navigate("/legacy", { replace: true });

    expect(router.state.location.pathname).toBe("/canonical");
    expect(router.state.historyAction).toBe("REPLACE");
  });
});
