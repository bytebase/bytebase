import { act, render } from "@testing-library/react";
import { StrictMode } from "react";
import { expect, test, vi } from "vitest";
import { ProjectSelect } from "./ProjectSelect";

const mocks = vi.hoisted(() => ({
  fetchProjectList: vi.fn(async () => ({ projects: [], nextPageToken: "" })),
  search: undefined as ((query: string) => Promise<void>) | undefined,
}));
vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({ fetchProjectList: mocks.fetchProjectList }),
  },
}));
vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({ onSearch }: { onSearch: (query: string) => Promise<void> }) => {
    mocks.search = onSearch;
    return null;
  },
}));
vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

test("forwards search cancellation through StrictMode replay, replacement, and unmount", async () => {
  const signals: AbortSignal[] = [];
  mocks.fetchProjectList.mockImplementation(
    async (params?: { signal?: AbortSignal }) => {
      expect(params?.signal).toBeInstanceOf(AbortSignal);
      signals.push(params!.signal!);
      return { projects: [], nextPageToken: "" };
    }
  );
  const view = render(
    <StrictMode>
      <ProjectSelect value="" onChange={() => {}} />
    </StrictMode>
  );
  await act(async () => {});
  expect(signals).toHaveLength(2);
  expect(signals[0].aborted).toBe(true);
  expect(signals[1].aborted).toBe(false);
  await act(() => mocks.search!("new"));
  expect(signals[1].aborted).toBe(true);
  expect(signals[2].aborted).toBe(false);
  view.unmount();
  expect(signals[2].aborted).toBe(true);
});
