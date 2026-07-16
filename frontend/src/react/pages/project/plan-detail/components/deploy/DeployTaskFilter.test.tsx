import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  StageSchema,
  TaskSchema,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { DeployTaskFilter } from "./DeployTaskFilter";

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/components/TaskStatusIcon", () => ({
  TaskStatusIcon: () => null,
}));

describe("DeployTaskFilter", () => {
  test("orders statuses by the canonical task status priority", () => {
    const stage = create(StageSchema, {
      tasks: [
        Task_Status.NOT_STARTED,
        Task_Status.DONE,
        Task_Status.FAILED,
        Task_Status.CANCELED,
        Task_Status.SKIPPED,
        Task_Status.PENDING,
        Task_Status.RUNNING,
      ].map((status, index) =>
        create(TaskSchema, { name: `tasks/${index}`, status })
      ),
    });

    render(
      <DeployTaskFilter
        onChange={vi.fn()}
        selectedStatuses={[]}
        stage={stage}
      />
    );

    expect(
      screen.getAllByRole("button").map((button) => button.textContent)
    ).toEqual([
      "task.status.failed1",
      "task.status.running1",
      "task.status.pending1",
      "task.status.canceled1",
      "task.status.not-started1",
      "task.status.done1",
      "task.status.skipped1",
    ]);
  });
});
