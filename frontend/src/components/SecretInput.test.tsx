import { fireEvent, render, screen, cleanup } from "@testing-library/react";
import { useState } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { SecretInput, SecretInputProvider, useHasPendingSecretEdits } from "./SecretInput";

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
afterEach(cleanup);

function Actions() {
  const pending = useHasPendingSecretEdits();
  return <button type="button" disabled={pending}>Save form</button>;
}
function Harness({ initial, allowEmpty = true, multiline = false }: { initial?: string; allowEmpty?: boolean; multiline?: boolean }) {
  const [value, setValue] = useState(initial);
  return <SecretInputProvider><SecretInput aria-label="Credential" value={value} onValueChange={setValue} allowEmpty={allowEmpty} multiline={multiline} /><Actions /><output data-testid="value">{value === undefined ? "unchanged" : JSON.stringify(value)}</output></SecretInputProvider>;
}
const click = (name: string) => fireEvent.click(screen.getByRole("button", { name }));
const type = (value: string) => fireEvent.change(screen.getByLabelText("Credential"), { target: { value } });

describe("SecretInput", () => {
  test("keeps the stored value until Done and preserves password whitespace", () => {
    render(<Harness />);
    expect((screen.getByLabelText("Credential") as HTMLInputElement).disabled).toBe(true);
    click("common.edit");
    expect(document.activeElement).toBe(screen.getByLabelText("Credential"));
    expect((screen.getByRole("button", { name: "Save form" }) as HTMLButtonElement).disabled).toBe(true);
    type("  replacement  ");
    expect(screen.getByTestId("value").textContent).toBe("unchanged");
    click("common.done");
    expect(screen.getByTestId("value").textContent).toBe('"  replacement  "');
    expect((screen.getByRole("button", { name: "Save form" }) as HTMLButtonElement).disabled).toBe(false);
    expect(document.activeElement).toBe(screen.getByRole("button", { name: "common.edit" }));
  });

  test("Cancel preserves a previously accepted replacement and does not submit", () => {
    const submit = vi.fn();
    render(<form onSubmit={submit}><Harness initial="accepted" /></form>);
    click("common.edit");
    type("discard me");
    fireEvent.keyDown(screen.getByLabelText("Credential"), { key: "Escape" });
    expect(screen.getByTestId("value").textContent).toBe('"accepted"');
    expect(submit).not.toHaveBeenCalled();
  });

  test("Done with blank input is an explicit clear; Cancel with blank input keeps", () => {
    render(<Harness />);
    click("common.edit");
    click("common.cancel");
    expect(screen.getByTestId("value").textContent).toBe("unchanged");
    click("common.edit");
    click("common.done");
    expect(screen.getByTestId("value").textContent).toBe('""');
    expect((screen.getByLabelText("Credential") as HTMLInputElement).placeholder).toBe("common.secret-input.empty");
  });

  test("required credentials cannot be cleared, including with Enter", () => {
    render(<Harness allowEmpty={false} />);
    click("common.edit");
    expect((screen.getByRole("button", { name: "common.done" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.keyDown(screen.getByLabelText("Credential"), { key: "Enter" });
    expect(screen.getByTestId("value").textContent).toBe("unchanged");
    type("new token");
    fireEvent.keyDown(screen.getByLabelText("Credential"), { key: "Enter" });
    expect(screen.getByTestId("value").textContent).toBe('"new token"');
  });

  test("new credentials are directly editable without a confirmation step", () => {
    const change = vi.fn();
    render(<SecretInput aria-label="Credential" isCreating value="" onValueChange={change} />);
    type("new password");
    expect(change).toHaveBeenCalledWith("new password");
    expect(screen.queryByRole("button", { name: "common.edit" })).toBeNull();
  });

  test("multiline input keeps line breaks and does not finish on Enter", () => {
    render(<Harness multiline />);
    click("common.edit");
    expect(screen.getByLabelText("Credential").tagName).toBe("TEXTAREA");
    type("first\nsecond\n");
    fireEvent.keyDown(screen.getByLabelText("Credential"), { key: "Enter" });
    expect(screen.getByTestId("value").textContent).toBe("unchanged");
    click("common.done");
    expect(screen.getByTestId("value").textContent).toBe(JSON.stringify("first\nsecond\n"));
  });

  test("resource changes discard an unfinished edit and release the form", () => {
    const onValueChange = vi.fn();
    const field = (key: string) => <SecretInputProvider><SecretInput resetKey={key} aria-label="Credential" value={undefined} onValueChange={onValueChange} /><Actions /></SecretInputProvider>;
    const view = render(field("admin"));
    click("common.edit");type("discard me");
    view.rerender(field("readonly"));
    expect(onValueChange).not.toHaveBeenCalled();
    expect((screen.getByRole("button", { name: "Save form" }) as HTMLButtonElement).disabled).toBe(false);
    expect(screen.queryByRole("button", { name: "common.done" })).toBeNull();
  });

  test("multiple active edits keep Save blocked until all are finished", () => {
    render(<SecretInputProvider><SecretInput aria-label="First" value={undefined} onValueChange={vi.fn()} /><SecretInput aria-label="Second" value={undefined} onValueChange={vi.fn()} /><Actions /></SecretInputProvider>);
    for (const button of screen.getAllByRole("button", { name: "common.edit" })) fireEvent.click(button);
    fireEvent.click(screen.getAllByRole("button", { name: "common.cancel" })[0]);
    expect((screen.getByRole("button", { name: "Save form" }) as HTMLButtonElement).disabled).toBe(true);
    click("common.cancel");
    expect((screen.getByRole("button", { name: "Save form" }) as HTMLButtonElement).disabled).toBe(false);
  });

  test("disabled secrets cannot open an edit", () => {
    const change=vi.fn();
    render(<SecretInput aria-label="Credential" value={undefined} disabled onValueChange={change} />);
    click("common.edit");
    expect(screen.queryByRole("button", { name: "common.done" })).toBeNull();
    expect(change).not.toHaveBeenCalled();
  });
});
