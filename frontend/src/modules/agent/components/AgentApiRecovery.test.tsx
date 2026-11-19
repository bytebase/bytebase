import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import * as apiAttempt from "../logic/apiAttempt";
import {
  type AgentApiAttempt,
  normalizeAgentApiAttempts,
} from "../logic/apiAttempt";
import { agentApiExecutionEvidence } from "../logic/apiExecution";
import { useAgentStore } from "../store/agent";
import { AgentApiRecovery } from "./AgentApiRecovery";

vi.mock("react-i18next", async (original) => ({
  ...(await original<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const marker = (): AgentApiAttempt => ({
  ...normalizeAgentApiAttempts([null], "chat")![0],
  id: "a",
  operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
  path: "/bytebase.v1.DatabaseService/UpdateDatabase",
});

function setup(attempt = marker()) {
  useAgentStore.setState({ currentChatId: "chat" });
  return render(<AgentApiRecovery attempt={attempt} />);
}

function expectTextOnlyWarning() {
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
  expect(screen.queryByRole("link")).not.toBeInTheDocument();
  expect(screen.queryByRole("group")).not.toBeInTheDocument();
  expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  expect(screen.getByText("agent.api-recovery.no-replay")).toBeVisible();
}

test("outcome warnings have no API-specific actions", () => {
  setup();
  expect(screen.getByText("agent.api-recovery.unknown")).toBeVisible();
  expectTextOnlyWarning();
});

test.each([
  ["fetch-failed", "unknown"],
  ["rpc-error", "failed-possible"],
  ["import-partial", "partial"],
  ["query-result-error", "possible"],
] as const)("the %s outcome remains visible without recovery actions", (reason, label) => {
  setup({ ...marker(), execution: agentApiExecutionEvidence(reason) });
  expect(screen.getByText(`agent.api-recovery.${label}`)).toBeVisible();
  expectTextOnlyWarning();
});

test.each([
  ["in-flight", "in-flight"],
  ["unavailable", "verifying"],
  ["unknown", "unknown"],
] as const)("dispatch liveness %s is informational only", async (status, label) => {
  vi.spyOn(apiAttempt, "getAgentApiAttemptStatus").mockResolvedValue(status);
  setup({ ...marker(), phase: "dispatching", execution: undefined });
  await screen.findByText(`agent.api-recovery.${label}`);
  expectTextOnlyWarning();
});

test("legacy review metadata does not add actions or change uncertainty", () => {
  setup({ ...marker(), reviewedExternallyAt: 42 });
  expect(screen.getByText("agent.api-recovery.unknown")).toBeVisible();
  expectTextOnlyWarning();
});

test("a different chat cannot show the warning", () => {
  setup();
  act(() => useAgentStore.setState({ currentChatId: "another-chat" }));
  expect(screen.queryByRole("region")).not.toBeInTheDocument();
});
