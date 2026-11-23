import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { MaskingRulePolicy_MaskingRuleSchema } from "@/types/proto-es/v1/org_policy_service_pb";
import { createMaskingRuleActionReceipt } from "../logic/actionReceipt";
import type { AgentActionObservation } from "../logic/types";
import { AgentActionReceiptActions } from "./AgentActionReceiptActions";

const mocks = vi.hoisted(() => ({check: vi.fn(), record: vi.fn(), agent: {} as Record<string, unknown>, app: {} as Record<string, unknown>}));
vi.mock("react-i18next", () => ({initReactI18next: {type: "3rdParty", init: () => {}}, useTranslation: () => ({t: (key: string) => key})}));
vi.mock("../logic/actionReceiptStatus", () => ({checkActionReceiptStatus: mocks.check}));
vi.mock("../store/agent", () => {
  const useAgentStore = <T,>(selector: (state: typeof mocks.agent) => T) => selector(mocks.agent);
  useAgentStore.getState = () => mocks.agent;
  return {useAgentStore};
});
vi.mock("@/stores/app", () => {
  const useAppStore = <T,>(selector: (state: typeof mocks.app) => T) => selector(mocks.app);
  useAppStore.getState = () => mocks.app;
  return {useAppStore};
});
const receipt = createMaskingRuleActionReceipt("workspaces/demo/policies/MASKING_RULE", create(MaskingRulePolicy_MaskingRuleSchema, {id: "rule-1", semanticType: "ssn", condition: {title: "Saved title", expression: "true"}}), "draft-1", 100)!;
beforeEach(() => {
  vi.clearAllMocks();
  const message = {id: "message-1", metadata: {actions: [{kind: "draft-masking-rule", draftId: "draft-1"}], actionReceipts: [receipt]}};
  mocks.agent = {workspaceName: "workspaces/demo", userEmail: "member@example.com", chats: [{id: "chat-1", workspaceName: "workspaces/demo"}], messagesByChatId: {"chat-1": [message]}, getChat: () => mocks.agent.chats && (mocks.agent.chats as object[])[0], captureApiScope: () => () => true, recordActionObservation: mocks.record};
  mocks.app = {workspace: {name: "workspaces/demo"}, currentUser: {workspace: "workspaces/demo", email: "member@example.com"}};
  mocks.record.mockReturnValue(message);
  mocks.check.mockResolvedValue({status: "CHANGED", observedAt: 200} satisfies AgentActionObservation);
});
afterEach(cleanup);
const show = () => render(<AgentActionReceiptActions chatId="chat-1" messageId="message-1" receipt={receipt} />);
const clickCheck = () => fireEvent.click(screen.getByRole("button", {name: "agent.submitted-access-grant.check-status"}));

test("shows saved outcome without polling and records only an explicit status check", async () => {
  show();
  expect(screen.getByText("Saved title")).toBeDefined();
  expect(screen.getByText("agent.action-receipt.saved")).toBeDefined();
  expect(mocks.check).not.toHaveBeenCalled();
  await act(async () => clickCheck());
  expect(mocks.check).toHaveBeenCalledExactlyOnceWith(receipt, expect.any(Function));
  expect(mocks.record).toHaveBeenCalledOnce();
  expect(screen.getByText("agent.action-receipt.changed")).toBeDefined();
});

test("keeps the save receipt when checking is unavailable", async () => {
  mocks.check.mockRejectedValue(new Error("permission denied"));
  show();
  await act(async () => clickCheck());
  expect(mocks.record.mock.calls[0]?.[3]).toMatchObject({status: "UNAVAILABLE"});
  expect(screen.getByRole("alert").textContent).toBe("agent.action-receipt.status-unavailable");
  expect(receipt.observation).toBeUndefined();
  expect(mocks.check).toHaveBeenCalledOnce();
});

test("discards checks after the source owner changes or unmounts", async () => {
  let resolve: ((value: AgentActionObservation) => void) | undefined;
  mocks.check.mockImplementation(() => new Promise((done) => {resolve = done;}));
  const {unmount} = show();
  clickCheck();
  unmount();
  await act(async () => resolve?.({status: "CHANGED", observedAt: 200}));
  expect(mocks.record).not.toHaveBeenCalled();
});
