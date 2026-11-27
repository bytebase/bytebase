import { create } from "@bufbuild/protobuf";
import { fireEvent, within } from "@testing-library/react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  type Database,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import { ExportSchemaField } from "./ExportSchemaField";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/DatabaseSelectOption", () => ({
  DatabaseSelectOption: ({ database }: { database: Database }) => (
    <span>{database.name}</span>
  ),
}));

vi.mock("@/components/DatabaseTargetDisplay", () => ({
  DatabaseTargetDisplay: ({ database }: { database: Database }) => (
    <span>{database.name}</span>
  ),
}));

describe("ExportSchemaField", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;
  const database = create(DatabaseSchema$, {
    name: "instances/example/databases/orders",
  });
  const label = "gitops.setup.export-schema.label";

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
  });

  test("names each database-picker group with its own field title", () => {
    act(() => {
      root.render(
        <>
          <ExportSchemaField candidates={[database]} value={undefined} onChange={() => {}} />
          <ExportSchemaField candidates={[database]} value={undefined} onChange={() => {}} />
        </>
      );
    });

    const groups = within(container).getAllByRole("group", { name: label });
    expect(groups).toHaveLength(2);
    const titleIds = groups.map((group) => group.getAttribute("aria-labelledby"));
    expect(new Set(titleIds).size).toBe(2);
    for (const [index, group] of groups.entries()) {
      expect(titleIds[index]).toBeTruthy();
      expect(document.getElementById(titleIds[index]!)).toHaveTextContent(label);
      expect(group.querySelector('[data-slot="combobox-trigger"]')).not.toBeNull();
    }
  });

  test("selects the canonical database through the existing picker", () => {
    const onChange = vi.fn();
    act(() => {
      root.render(<ExportSchemaField candidates={[database]} value={undefined} onChange={onChange} />);
    });
    const group = within(container).getByRole("group", { name: label });
    const trigger = group.querySelector('[data-slot="combobox-trigger"]')!;
    act(() => fireEvent.click(trigger));
    const option = within(document.body).getByRole("option", { name: database.name });
    act(() => fireEvent.click(option));
    expect(onChange).toHaveBeenCalledExactlyOnceWith(database.name);
  });

  test("keeps an empty picker unavailable while displaying its explanation", () => {
    const onChange = vi.fn();
    act(() => {
      root.render(<ExportSchemaField candidates={[]} value={undefined} onChange={onChange} />);
    });
    const group = within(container).getByRole("group", { name: label });
    expect(group).toHaveTextContent("gitops.setup.export-schema.none-supported");
    expect(group).toHaveTextContent("gitops.setup.export-schema.description");
    act(() => fireEvent.click(group.querySelector('[data-slot="combobox-trigger"]')!));
    expect(within(document.body).queryByRole("listbox")).toBeNull();
    expect(onChange).not.toHaveBeenCalled();
  });
});
