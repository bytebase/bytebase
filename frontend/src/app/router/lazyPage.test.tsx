import { render, screen, waitFor } from "@testing-library/react";
import { useRef } from "react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { describe, expect, test } from "vitest";
import { lazyPage } from "./lazyPage";

describe("lazyPage persistent parent routes", () => {
  test("forwards leaf params without remounting the parent page", async () => {
    let mountCount = 0;
    const Page = (props: Record<string, unknown>) => {
      const mountId = useRef<number | undefined>(undefined);
      if (mountId.current === undefined) {
        mountCount += 1;
        mountId.current = mountCount;
      }
      return (
        <div>
          <span data-testid="mount-id">{mountId.current}</span>
          <span data-testid="route-name">{String(props.routeName)}</span>
          <span data-testid="route-hash">{String(props.routeHash)}</span>
          <span data-testid="spec-id">{String(props.specId ?? "")}</span>
          <span data-testid="stage-id">{String(props.stageId ?? "")}</span>
          <span data-testid="task-id">{String(props.taskId ?? "")}</span>
        </div>
      );
    };
    const router = createMemoryRouter(
      [
        {
          path: "/projects/:projectId/plans/:planId",
          lazy: lazyPage(
            async () => ({ Page }),
            (module) => module.Page
          ),
          children: [
            { index: true, handle: { name: "plan" } },
            { path: "specs/:specId", handle: { name: "plan.spec" } },
            {
              path: "rollout/stages/:stageId",
              handle: { name: "plan.rollout.stage" },
            },
            {
              path: "rollout/stages/:stageId/tasks/:taskId",
              handle: { name: "plan.rollout.stage.task" },
            },
          ],
        },
      ],
      { initialEntries: ["/projects/p/plans/1"] }
    );

    render(<RouterProvider router={router} />);
    await screen.findByText("plan");
    expect(screen.getByTestId("mount-id")).toHaveTextContent("1");

    await router.navigate("/projects/p/plans/1/specs/spec-2#result-7");
    await waitFor(() =>
      expect(screen.getByTestId("route-name")).toHaveTextContent("plan.spec")
    );
    expect(screen.getByTestId("spec-id")).toHaveTextContent("spec-2");
    expect(screen.getByTestId("route-hash")).toHaveTextContent("#result-7");
    expect(screen.getByTestId("mount-id")).toHaveTextContent("1");

    await router.navigate("/projects/p/plans/1/rollout/stages/prod");
    await waitFor(() =>
      expect(screen.getByTestId("route-name")).toHaveTextContent(
        "plan.rollout.stage"
      )
    );
    expect(screen.getByTestId("stage-id")).toHaveTextContent("prod");
    expect(screen.getByTestId("mount-id")).toHaveTextContent("1");

    await router.navigate(
      "/projects/p/plans/1/rollout/stages/prod/tasks/42?taskRunId=7#log"
    );
    await waitFor(() =>
      expect(screen.getByTestId("route-name")).toHaveTextContent(
        "plan.rollout.stage.task"
      )
    );
    expect(screen.getByTestId("stage-id")).toHaveTextContent("prod");
    expect(screen.getByTestId("task-id")).toHaveTextContent("42");
    expect(screen.getByTestId("route-hash")).toHaveTextContent("#log");
    expect(screen.getByTestId("mount-id")).toHaveTextContent("1");

    await router.navigate(-1);
    await waitFor(() =>
      expect(screen.getByTestId("route-name")).toHaveTextContent(
        "plan.rollout.stage"
      )
    );
    expect(screen.getByTestId("task-id")).toHaveTextContent("");
    expect(screen.getByTestId("mount-id")).toHaveTextContent("1");

    await router.navigate(1);
    await waitFor(() =>
      expect(screen.getByTestId("task-id")).toHaveTextContent("42")
    );
    expect(screen.getByTestId("mount-id")).toHaveTextContent("1");
    expect(mountCount).toBe(1);
  });
});
