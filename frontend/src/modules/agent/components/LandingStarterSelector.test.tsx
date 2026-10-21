import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import type { LandingStarterId } from "../logic/landingStarters";

const prompts: Record<LandingStarterId, string> = {
  "workspace-setup-first": "What should I set up first?",
  "workspace-concepts": "Help me understand Projects, Instances, and Databases",
  "workspace-governance": "How does Bytebase govern database changes?",
  "workspace-attention": "What needs my attention?",
  "workspace-recent": "What changed recently across my projects?",
  "project-connect": "How do I connect a database to this project?",
  "project-organize": "How should I organize databases and environments for this project?",
  "project-next": "What can I do next?",
  "project-propose-change": "How do I propose a database change?",
  "project-open-blockers": "What needs my attention in this project?",
  "project-open-recent": "What changed recently in this project?",
  "project-collaborate": "How do I invite teammates to this project?",
  "project-access": "What should I set up before collaborating on this project?",
  "project-review": "How are database changes reviewed in this project?",
};

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const id = key.replace("agent.landing-starters.", "") as LandingStarterId;
      return prompts[id] ?? key;
    },
  }),
}));

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuTrigger: ({ render }: { render: React.ReactElement }) => render,
  DropdownMenuContent: ({ children, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
    <div {...props}>{children}</div>
  ),
  DropdownMenuItem: ({ children, ...props }: React.HTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>{children}</button>
  ),
}));

import { LandingStarterSelector } from "./LandingStarterSelector";

afterEach(() => {
  document.body.replaceChildren();
});

const render = (props: React.ComponentProps<typeof LandingStarterSelector>) => {
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  act(() => root.render(<LandingStarterSelector {...props} />));
  return { container, unmount: () => act(() => root.unmount()) };
};

describe("LandingStarterSelector", () => {
  test("sends the highest-ranked prompt directly", () => {
    const onSelect = vi.fn();
    const view = render({
      starterIds: ["project-next", "project-propose-change"],
      disabled: false,
      onSelect,
    });

    const primary = view.container.querySelector<HTMLButtonElement>(
      '[data-testid="landing-starter-primary"]'
    );
    expect(primary?.textContent).toBe("What can I do next?");
    act(() => primary?.click());
    expect(onSelect).toHaveBeenCalledWith("project-next");
    view.unmount();
  });

  test("uses setup-oriented prompts for a solo Project", () => {
    const view = render({
      starterIds: ["project-collaborate", "project-access", "project-review"],
      disabled: false,
      onSelect: vi.fn(),
    });

    expect(
      view.container.querySelector('[data-testid="landing-starter-primary"]')
        ?.textContent
    ).toBe("How do I invite teammates to this project?");
    expect(
      view.container.querySelector('[data-starter-id="project-access"]')
        ?.textContent
    ).toBe("What should I set up before collaborating on this project?");
    view.unmount();
  });

  test("only renders overflow alternatives when there is more than one prompt", () => {
    const single = render({
      starterIds: ["project-next"],
      disabled: false,
      onSelect: vi.fn(),
    });
    expect(
      single.container.querySelector('[data-testid="landing-starter-overflow"]')
    ).toBeNull();
    single.unmount();

    const multiple = render({
      starterIds: ["project-next", "project-propose-change"],
      disabled: false,
      onSelect: vi.fn(),
    });
    expect(
      multiple.container.querySelector('[data-testid="landing-starter-overflow"]')
    ).not.toBeNull();
    multiple.unmount();
  });

  test("vertically centers the overflow control with the prompt", () => {
    const view = render({
      starterIds: ["project-next", "project-propose-change"],
      disabled: false,
      onSelect: vi.fn(),
    });

    expect(view.container.firstElementChild?.className).toContain(
      "items-center"
    );
    view.unmount();
  });

  test("sends an alternative from the compact menu", () => {
    const onSelect = vi.fn();
    const view = render({
      starterIds: ["project-next", "project-organize", "project-propose-change"],
      disabled: false,
      onSelect,
    });

    const alternative = view.container.querySelector<HTMLButtonElement>(
      '[data-starter-id="project-propose-change"]'
    );
    act(() => alternative?.click());
    expect(onSelect).toHaveBeenCalledWith("project-propose-change");
    view.unmount();
  });

  test("prevents selection while the agent is busy", () => {
    const onSelect = vi.fn();
    const view = render({
      starterIds: ["project-next", "project-propose-change"],
      disabled: true,
      onSelect,
    });

    expect(
      view.container.querySelector<HTMLButtonElement>(
        '[data-testid="landing-starter-primary"]'
      )?.disabled
    ).toBe(true);
    expect(
      view.container.querySelector<HTMLButtonElement>(
        '[data-testid="landing-starter-overflow"]'
      )?.disabled
    ).toBe(true);
    expect(
      view.container.querySelector<HTMLButtonElement>(
        '[data-starter-id="project-propose-change"]'
      )?.disabled
    ).toBe(true);
    view.unmount();
  });

  test("keeps many wrapped alternatives inside a scrollable menu", () => {
    const view = render({
      starterIds: [
        "project-organize",
        "project-next",
        "project-propose-change",
        "project-connect",
      ],
      disabled: false,
      onSelect: vi.fn(),
    });

    const menu = view.container.querySelector('[data-testid="landing-starter-menu"]');
    expect(menu?.className).toContain("max-h-64");
    expect(menu?.className).toContain("overflow-y-auto");
    expect(
      view.container.querySelector('[data-starter-id="project-propose-change"]')
        ?.className
    ).toContain("whitespace-normal");
    expect(
      view.container.querySelectorAll("[data-starter-id]")
    ).toHaveLength(3);
    view.unmount();
  });
});
