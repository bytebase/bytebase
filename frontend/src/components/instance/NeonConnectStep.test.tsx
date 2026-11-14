import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { NeonConnectStep } from "./NeonConnectStep";

const t = (key: string, values?: Record<string, unknown>) =>
  values ? `${key}${JSON.stringify(values)}` : key;
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t }) }));

afterEach(cleanup);

const failure = (kind: string, retryAfterSeconds = 0) =>
  ({ kind, retryAfterSeconds, message: "" }) as Parameters<
    typeof NeonConnectStep
  >[0]["failure"];

function renderStep(props: Partial<Parameters<typeof NeonConnectStep>[0]> = {}) {
  const handlers = {
    onApiKeyChange: vi.fn(),
    onProjectIdChange: vi.fn(),
    onSubmit: vi.fn(),
  };
  render(
    <NeonConnectStep
      apiKey="napi_example"
      projectId=""
      busy={false}
      destination="Workspace"
      {...handlers}
      {...props}
    />
  );
  return handlers;
}

test("masks the API key behind a reveal control and says what is saved", () => {
  renderStep();
  const key = screen.getByLabelText("instance.neon.connect.api-key");
  expect(key).toHaveAttribute("type", "password");
  fireEvent.click(
    screen.getByRole("button", { name: "common.toggle-password-visibility" })
  );
  expect(key).toHaveAttribute("type", "text");
  expect(
    screen.getByText("instance.neon.connect.not-saved-body")
  ).toBeVisible();
  expect(
    screen.getByText("instance.neon.connect.credentials-saved-body")
  ).toBeVisible();
  expect(
    screen.getByText("instance.neon.connect.project-id-help")
  ).toBeVisible();
});

test("puts a rejected key on the key field", () => {
  renderStep({ failure: failure("key-invalid") });
  const key = screen.getByLabelText("instance.neon.connect.api-key");
  expect(key).toHaveAttribute("aria-invalid", "true");
  expect(key).toHaveAccessibleDescription(
    "instance.neon.connect.error.key-invalid"
  );
  expect(key).toHaveValue("napi_example");
});

test("asks for the Neon project ID when the key cannot list Neon projects", () => {
  renderStep({ failure: failure("project-id-required") });
  expect(
    screen.getByLabelText(/instance.neon.connect.project-id/)
  ).toHaveAccessibleDescription(
    "instance.neon.connect.error.project-id-required"
  );
  expect(
    screen.getByLabelText("instance.neon.connect.api-key")
  ).not.toHaveAttribute("aria-invalid");
});

test("states when a rate limit ends as a clock time, so the alert does not change every second", () => {
  renderStep({
    failure: failure("rate-limited", 25),
    retryAt: new Date(2030, 0, 1, 14, 32, 10).getTime(),
  });
  const alert = screen.getByRole("alert");
  expect(alert).toHaveTextContent(
    'instance.neon.connect.error.rate-limited-wait{"time":"14:32:10"}'
  );
  // The shared warning color is too faint for body text.
  expect(alert.className.split(" ")).toContain("text-warning-hover");
  expect(alert.className.split(" ")).not.toContain("text-warning");
});

test("Enter in a field starts discovery", () => {
  const { onSubmit } = renderStep();
  const form = screen
    .getByLabelText("instance.neon.connect.api-key")
    .closest("form");
  expect(form).not.toBeNull();
  fireEvent.submit(form as HTMLFormElement);
  expect(onSubmit).toHaveBeenCalledOnce();
});

test("offers retry after an outage or deadline", () => {
  const { onSubmit } = renderStep({ failure: failure("unavailable") });
  fireEvent.click(
    screen.getByRole("button", { name: "instance.neon.connect.try-again" })
  );
  expect(onSubmit).toHaveBeenCalledOnce();
});

test("names the destination when Bytebase denies adding instances", () => {
  renderStep({ failure: failure("permission-denied") });
  expect(
    screen.getByText(
      'instance.neon.connect.error.permission-body{"destination":"Workspace"}'
    )
  ).toBeVisible();
});
