import { create } from "@bufbuild/protobuf";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, test } from "vitest";
import i18n from "@/lib/i18n";
import { Task_ReleaseStateSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { DeployTaskPlannedFiles } from "./DeployTaskPlannedFiles";

beforeAll(async () => {
  await i18n.changeLanguage("en-US");
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) =>
      ({
        matches: false,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      }) as unknown as MediaQueryList,
  });
});
afterEach(cleanup);

const files = ["001", "002", "003"].map((version) => ({
  version,
  path: `migrations/${version}.sql`,
}));
const state = create(Task_ReleaseStateSchema, {
  pendingRanges: [{ start: 1, end: 3 }],
  pendingCount: 2,
  appliedCount: 1,
});

describe("DeployTaskPlannedFiles", () => {
  test("a task that has not run lists what it will apply, as of rollout creation", () => {
    render(
      <DeployTaskPlannedFiles
        computedAgo="5 minutes ago"
        files={files}
        skipped={false}
        state={state}
      />
    );
    expect(screen.getByText(/the run re-checks the database/)).toBeTruthy();
    expect(screen.getByTestId("spine-file-002").textContent).toContain(
      "Will apply"
    );
    expect(screen.queryByTestId("deploy-task-planned-toggle")).toBeNull();
  });

  test("a task skipped before it ran keeps the plan behind a closed toggle, and nothing reads as about to apply", () => {
    render(
      <DeployTaskPlannedFiles
        computedAgo="5 minutes ago"
        files={files}
        skipped
        state={state}
      />
    );
    const toggle = screen.getByTestId("deploy-task-planned-toggle");
    expect(toggle.textContent).toBe(
      "Planned at rollout creation · 2 files to apply"
    );
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByTestId("spine-file-002")).toBeNull();

    act(() => fireEvent.click(toggle));
    expect(screen.getByTestId("spine-file-002").textContent).toContain(
      "Not applied"
    );
    expect(document.body.textContent).not.toContain("Will apply");
    expect(document.body.textContent).not.toContain("re-checks");
  });
});
