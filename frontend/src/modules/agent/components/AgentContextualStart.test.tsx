import { fireEvent, render, screen } from "@testing-library/react";
import { createInstance } from "i18next";
import { useTranslation } from "react-i18next";
import { describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import type { AgentPageResource } from "../logic/chatContext";
import type { AgentResourceReference } from "../logic/types";
import { AgentContextualStart } from "./AgentContextualStart";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
}));

const renderContext = (
  page: AgentPageResource,
  reference?: AgentResourceReference
) => {
  const onStart = vi.fn();
  render(
    <AgentContextualStart
      page={page}
      title="Masking settings"
      reference={reference}
      loading={false}
      canStart
      hasCurrentChat={false}
      onStart={onStart}
      onContinue={vi.fn()}
    />
  );
  return onStart;
};

test("shows Continue last chat as the localized primary action", async () => {
  const i18n = createInstance();
  await i18n.init({
    lng: "en-US",
    resources: { "en-US": { translation: enUS } },
  });
  vi.mocked(useTranslation).mockReturnValueOnce({
    t: i18n.t,
  } as ReturnType<typeof useTranslation>);
  const onContinue = vi.fn();
  const onStart = vi.fn();
  render(
    <AgentContextualStart
      page={{
        kind: "project",
        name: "projects/sample",
        project: "projects/sample",
      }}
      title="Sample project"
      loading={false}
      canStart
      hasCurrentChat
      onStart={onStart}
      onContinue={onContinue}
    />
  );

  const button = screen.getByRole("button", {
    name: "Continue last chat",
  });
  expect(button).toHaveClass("bg-accent", "text-accent-text");
  fireEvent.click(button);
  expect(onContinue).toHaveBeenCalledOnce();
  expect(onStart).not.toHaveBeenCalled();
});

test("hides Continue when there is no eligible chat", () => {
  renderContext({ kind: "global-masking" });
  expect(
    screen.queryByRole("button", {
      name: "agent.contextual-start.continue-chat",
    })
  ).toBeNull();
});

describe("AgentContextualStart masking contexts", () => {
  test("shows three rule-editor tasks without introducing another context label", () => {
    const onStart = vi.fn();
    render(
      <AgentContextualStart
        page={{ kind: "global-masking" }}
        title="Global Masking"
        maskingRuleEditor
        loading={false}
        canStart
        hasCurrentChat={false}
        onStart={onStart}
        onContinue={vi.fn()}
      />
    );
    const prompts = [
      "agent.contextual-start.masking-rule-configure",
      "agent.contextual-start.masking-rule-effect",
      "agent.contextual-start.masking-rule-review",
    ];
    expect(screen.getByText("Global Masking")).toBeTruthy();
    expect(screen.queryByText("agent.contextual-start.masking-global-effect")).toBeNull();
    expect(screen.getAllByRole("button")).toHaveLength(3);
    for (const prompt of prompts) {
      fireEvent.click(screen.getByRole("button", { name: prompt }));
      expect(onStart).toHaveBeenLastCalledWith(prompt, undefined);
    }
    expect(screen.queryByText(/^Context:/)).toBeNull();
  });

  test.each([
    [
      { kind: "global-masking" },
      "agent.contextual-start.masking-global-heading",
      "agent.contextual-start.masking-global-effect",
    ],
    [
      { kind: "semantic-types" },
      "agent.contextual-start.masking-semantic-heading",
      "agent.contextual-start.masking-semantic-algorithm",
    ],
    [
      { kind: "data-classification" },
      "agent.contextual-start.masking-classification-heading",
      "agent.contextual-start.masking-classification-effect",
    ],
    [
      { kind: "database-catalog", project: "projects/sample", names: ["projects/sample/instances/primary/databases/app"] },
      "agent.contextual-start.masking-catalog-heading",
      "agent.contextual-start.masking-catalog-visibility",
    ],
    [
      { kind: "masking-exemption", project: "projects/sample" },
      "agent.contextual-start.masking-exemption-heading",
      "agent.contextual-start.masking-exemption-effect",
    ],
  ] as const)("shows tailored help for %j", (page, heading, suggestion) => {
    renderContext(
      page as AgentPageResource,
      page.kind === "database-catalog"
        ? {
            kind: "database",
            name: page.names[0],
            project: page.project,
            label: "App database",
          }
        : undefined
    );
    expect(screen.getByText(heading)).toBeTruthy();
    expect(screen.getByText(suggestion)).toBeTruthy();
  });

  test("sends the selected masking question with the existing page context", () => {
    const onStart = renderContext({ kind: "global-masking" } as AgentPageResource);
    fireEvent.click(
      screen.getByRole("button", {
        name: "agent.contextual-start.masking-global-effect",
      })
    );
    expect(onStart).toHaveBeenCalledWith(
      "agent.contextual-start.masking-global-effect",
      undefined
    );
  });
});
