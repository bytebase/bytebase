import type { ReactNode } from "react";
import type { Environment } from "@/types/v1/environment";

/** Two environments, one of them protected, for any test that needs a list. */
export const STUB_ENVIRONMENTS = [
  {
    id: "staging",
    name: "environments/staging",
    title: "Staging",
    tags: {},
    color: "",
  },
  {
    id: "prod",
    name: "environments/prod",
    title: "Prod",
    tags: { protected: "protected" },
    color: "",
  },
] as unknown as Environment[];

/**
 * A stand-in for `EnvironmentSelect` that exposes what a form test drives:
 * the current value, one row per environment with its suffix, and a target
 * that picks Staging, without the store the real picker reads from.
 * Use as `vi.mock("@/components/EnvironmentSelect", async () =>
 *   (await import("@/test-utils/environmentSelectStub")).environmentSelectStub())`.
 */
export const environmentSelectStub = () => ({
  EnvironmentSelect: ({
    value,
    onChange,
    placeholder,
    renderSuffix,
  }: {
    value: string[];
    onChange: (next: string[]) => void;
    placeholder?: string;
    renderSuffix?: (environment: Environment) => ReactNode;
  }) => (
    <div data-testid="env-multi-select" data-value={value.join(",")}>
      <span>{placeholder}</span>
      {STUB_ENVIRONMENTS.map((environment) => (
        <span key={environment.name} data-testid="env-option">
          {environment.title}
          {renderSuffix?.(environment)}
        </span>
      ))}
      <span
        data-testid="pick-staging"
        onClick={() => onChange(["environments/staging"])}
      />
    </div>
  ),
});

/**
 * A stand-in for `EnvironmentBadge` that shows only the name it was given,
 * so a chip assertion reads the resolved environment and nothing else.
 * Use as `vi.mock("@/components/EnvironmentLabel", async () =>
 *   (await import("@/test-utils/environmentSelectStub")).environmentBadgeStub())`.
 */
export const environmentBadgeStub = () => ({
  EnvironmentBadge: ({ environment }: { environment: { name: string } }) => (
    <span data-testid="env-label">{environment.name}</span>
  ),
});
