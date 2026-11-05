import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { AgentPageResource } from "../logic/chatContext";
import type { AgentResourceReference } from "../logic/types";
import { AgentContextualStart } from "./AgentContextualStart";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
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

describe("AgentContextualStart masking contexts", () => {
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
