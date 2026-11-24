import { create } from "@bufbuild/protobuf";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import {
  TaskRun_Status,
  TaskRunSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunErrorAlert } from "./TaskRunErrorAlert";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

afterEach(cleanup);

test("places diagnostic help after the full error inside the alert", () => {
  render(
    <TaskRunErrorAlert
      taskRun={create(TaskRunSchema, {
        status: TaskRun_Status.FAILED,
        detail: "ERROR: missing table\nDETAIL: execution stopped",
      })}
      action={<button type="button">Ask Agent</button>}
    />
  );
  const alert = screen.getByRole("alert");
  const action = within(alert).getByRole("button", { name: "Ask Agent" });
  const error = within(alert).getByText(/ERROR: missing table/);
  expect(error.textContent).toBe("ERROR: missing table\nDETAIL: execution stopped");
  expect(error.compareDocumentPosition(action) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
});

test("keeps failed-run help inside an alert even without an error detail", () => {
  render(
    <TaskRunErrorAlert
      taskRun={create(TaskRunSchema, { status: TaskRun_Status.FAILED })}
      action={<button type="button">Ask Agent</button>}
    />
  );
  expect(within(screen.getByRole("alert")).getByRole("button", { name: "Ask Agent" })).toBeTruthy();
});

test("does not invent an empty error alert without detail or help", () => {
  render(<TaskRunErrorAlert taskRun={create(TaskRunSchema, { status: TaskRun_Status.FAILED })} />);
  expect(screen.queryByRole("alert")).toBeNull();
});

test("does not label a completed run as an error", () => {
  render(
    <TaskRunErrorAlert
      taskRun={create(TaskRunSchema, { status: TaskRun_Status.DONE, detail: "Old error" })}
      action={<button type="button">Ask Agent</button>}
    />
  );
  expect(screen.queryByRole("alert")).toBeNull();
});
