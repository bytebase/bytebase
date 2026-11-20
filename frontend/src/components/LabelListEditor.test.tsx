import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import { LabelListEditor } from "./LabelListEditor";

vi.mock("react-i18next", async () =>
  (await import("@/test-utils/i18n")).reactI18nextStub()
);
test("exposes only exact editing controls and releases detached refs", () => {
  const ref = vi.fn();
  const props = {
    kvList: [{ key: "team", value: "ops" }],
    readonly: false,
    onChange: vi.fn(),
    controlRef: ref,
  };
  const view = render(<LabelListEditor {...props} />);
  expect(ref).toHaveBeenCalledWith(
    { kind: "key", row: 0 },
    screen.getByDisplayValue("team")
  );
  expect(ref).toHaveBeenCalledWith(
    { kind: "value", row: 0 },
    screen.getByDisplayValue("ops")
  );
  expect(ref).toHaveBeenCalledWith(
    { kind: "remove", row: 0 },
    screen.getByRole("button", { name: "common.delete" })
  );
  expect(ref).toHaveBeenCalledWith(
    { kind: "add" },
    screen.getByRole("button", { name: "label.add-label" })
  );
  ref.mockClear();
  view.rerender(<LabelListEditor {...props} />);
  expect(ref).not.toHaveBeenCalled();
  view.unmount();
  expect(ref.mock.calls.map(([, element]) => element)).toEqual([
    null,
    null,
    null,
    null,
  ]);
});
