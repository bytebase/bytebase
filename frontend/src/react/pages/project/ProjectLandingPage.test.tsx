import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  replace: vi.fn(),
}));

vi.mock("@/router", () => ({
  router: {
    replace: mocks.replace,
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_ISSUES: "project.issues",
}));

import { ProjectLandingPage } from "./ProjectLandingPage";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  mocks.replace.mockClear();
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

describe("ProjectLandingPage", () => {
  test("redirects project landing to issues", async () => {
    await act(async () => {
      root.render(<ProjectLandingPage projectId="prod" />);
    });

    expect(mocks.replace).toHaveBeenCalledWith({
      name: "project.issues",
      params: { projectId: "prod" },
    });
  });
});
