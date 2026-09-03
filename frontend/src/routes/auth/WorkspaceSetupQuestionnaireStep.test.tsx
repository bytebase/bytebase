import { fireEvent } from "@testing-library/react";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import type {
  GuideScenarioId,
  GuideWorkspaceUsage,
} from "@/modules/workspace-setup-guide/types";
import { WorkspaceSetupQuestionnaireStep } from "./WorkspaceSetupQuestionnaireStep";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({
    t: (key: string) =>
      (
        {
          "settings.profile.setup-scenario.outcome-title":
            "What would you like to do with Bytebase?",
          "settings.profile.setup-scenario.create-database-change.title":
            "Create a database change",
          "settings.profile.setup-scenario.create-database-change.description":
            "Define a change and create its issue.",
          "settings.profile.setup-scenario.query-data.title": "Query data",
          "settings.profile.setup-scenario.query-data.description":
            "Open SQL Editor and run a statement.",
          "settings.profile.setup-scenario.workspace-usage.title":
            "Who will use Bytebase with you?",
          "settings.profile.setup-scenario.workspace-usage.team.title":
            "My team",
          "settings.profile.setup-scenario.workspace-usage.team.description":
            "I will work with other people.",
          "settings.profile.setup-scenario.workspace-usage.solo.title":
            "Just me",
          "settings.profile.setup-scenario.workspace-usage.solo.description":
            "I will use Bytebase on my own.",
          "settings.profile.setup-scenario.continue": "Continue",
        } as Record<string, string>
      )[key] ?? key,
  }),
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  act(() => root.render(element));
  return {
    container,
    unmount: () => act(() => root.unmount()),
  };
};

const renderQuestionnaire = (
  props: Partial<{
    scenarioValue: GuideScenarioId;
    workspaceUsageValue: GuideWorkspaceUsage;
    onScenarioChange: (value: GuideScenarioId) => void;
    onWorkspaceUsageChange: (value: GuideWorkspaceUsage) => void;
    onContinue: () => void;
  }> = {}
) =>
  renderIntoContainer(
    <WorkspaceSetupQuestionnaireStep
      scenarioValue={props.scenarioValue}
      workspaceUsageValue={props.workspaceUsageValue}
      onScenarioChange={props.onScenarioChange ?? vi.fn()}
      onWorkspaceUsageChange={props.onWorkspaceUsageChange ?? vi.fn()}
      onContinue={props.onContinue ?? vi.fn()}
    />
  );

describe("WorkspaceSetupQuestionnaireStep", () => {
  beforeEach(() => vi.clearAllMocks());

  test("uses a goal question that names Bytebase", () => {
    expect(
      enUS.settings.profile["setup-scenario"]["outcome-title"]
    ).toBe("What would you like to do with Bytebase?");
  });

  test("starts with both questions unanswered and allows Continue", () => {
    const page = renderQuestionnaire();
    const radios = [...page.container.querySelectorAll("[role='radio']")];
    const buttons = [...page.container.querySelectorAll("button")];

    expect(radios).toHaveLength(4);
    expect(radios.every((radio) => radio.getAttribute("aria-checked") === "false"))
      .toBe(true);
    expect(buttons).toHaveLength(1);
    expect(buttons[0]).toHaveTextContent("Continue");
    expect(buttons[0]).toBeEnabled();
    expect(page.container.textContent).not.toContain("Skip");
    expect(page.container.querySelector("h1")).toHaveTextContent(
      "What would you like to do with Bytebase?"
    );
    expect(page.container.textContent).not.toContain(
      "Tell us about your setup"
    );
    expect(page.container.textContent).not.toContain(
      "We will tailor your getting-started guide."
    );
    page.unmount();
  });

  test("reports each answer independently", () => {
    const onScenarioChange = vi.fn();
    const onWorkspaceUsageChange = vi.fn();
    const page = renderQuestionnaire({
      onScenarioChange,
      onWorkspaceUsageChange,
    });
    const labels = [...page.container.querySelectorAll("label")];

    act(() =>
      fireEvent.click(
        labels.find((label) => label.textContent?.includes("Query data"))!
      )
    );
    act(() =>
      fireEvent.click(
        labels.find((label) => label.textContent?.includes("My team"))!
      )
    );

    expect(onScenarioChange).toHaveBeenCalledWith("query-data");
    expect(onWorkspaceUsageChange).toHaveBeenCalledWith("team");
    page.unmount();
  });

  test("continues without requiring either answer", () => {
    const onContinue = vi.fn();
    const page = renderQuestionnaire({ onContinue });

    act(() => fireEvent.click(page.container.querySelector("button")!));

    expect(onContinue).toHaveBeenCalledOnce();
    page.unmount();
  });
});
