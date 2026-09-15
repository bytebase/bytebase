import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { SecretInput, SecretInputProvider, useHasInvalidSecretInputs } from "./SecretInput";
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
afterEach(cleanup);
function Actions() {
  const invalid = useHasInvalidSecretInputs();
  return <button disabled={invalid}>Update</button>;
}
function Harness({ allowEmpty = true, multiline = false }: { allowEmpty?: boolean; multiline?: boolean }) {
  const [value, setValue] = useState<string>();
  return <SecretInputProvider><SecretInput aria-label="Credential" value={value} onValueChange={setValue} allowEmpty={allowEmpty} multiline={multiline} /><Actions /><output data-testid="value">{value === undefined ? "unchanged" : JSON.stringify(value)}</output></SecretInputProvider>;
}
const input = () => screen.getByLabelText("Credential") as HTMLInputElement;
const type = (value: string) => fireEvent.change(input(), { target: { value } });
describe("SecretInput", () => {
  test("focus preserves the secret; typing replaces it without copying the mask", () => {
    render(<Harness />);
    expect(input().value).toBe("");
    expect(input().placeholder).toBe("common.secret-input.stored");
    fireEvent.focus(input());
    expect(screen.getByTestId("value").textContent).toBe("unchanged");
    type("  replacement  ");
    expect(input().value).toBe("  replacement  ");
    expect(input().type).toBe("password");
    expect(screen.getByTestId("value").textContent).toBe('"  replacement  "');
    expect(screen.queryByRole("button", { name: "common.done" })).toBeNull();
  });
  test("Clear explicitly empties an untouched secret without typing", () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole("button", { name: "common.clear" }));
    expect(screen.getByTestId("value").textContent).toBe('""');
    expect(input().placeholder).toBe("");
    expect(document.activeElement).toBe(input());
    expect(screen.queryByRole("button", { name: "common.clear" })).toBeNull();
  });
  test("deleting in the empty stored-value input leaves it unchanged", () => {
    render(<Harness />);
    fireEvent.keyDown(input(), { key: "Backspace" });
    expect(screen.getByTestId("value").textContent).toBe("unchanged");
  });
  test("clearing a required secret blocks Update until a replacement is entered", () => {
    render(<Harness allowEmpty={false} />);
    const update = screen.getByRole("button", { name: "Update" }) as HTMLButtonElement;
    expect(update.disabled).toBe(false);
    expect(screen.queryByRole("button", { name: "common.clear" })).toBeNull();
    type("replacement");
    type("");
    expect(update.disabled).toBe(true);
    expect(input().required).toBe(true);
    type("replacement");
    expect(update.disabled).toBe(false);
  });
  test("the eye reveals only a replacement and never submits", () => {
    const submit = vi.fn();
    render(<form onSubmit={submit}><Harness /></form>);
    expect(screen.queryByRole("button", { name: "common.toggle-password-visibility" })).toBeNull();
    type("replacement");
    fireEvent.click(screen.getByRole("button", { name: "common.toggle-password-visibility" }));
    expect(input().type).toBe("text");
    expect(input().value).toBe("replacement");
    expect(submit).not.toHaveBeenCalled();
  });
  test("form reset restores the stored mask and resets reveal state", () => {
    const field = (value: string | undefined, key: number) => <SecretInputProvider resetKey={key}><SecretInput aria-label="Credential" value={value} onValueChange={vi.fn()} /><Actions /></SecretInputProvider>;
    const view = render(field("replacement", 0));
    fireEvent.click(screen.getByRole("button", { name: "common.toggle-password-visibility" }));
    view.rerender(field(undefined, 1));
    expect(input().value).toBe("");
    expect(input().placeholder).toBe("common.secret-input.stored");
    expect(input().type).toBe("password");
  });
  test("multiline replacements retain line breaks", () => {
    render(<Harness multiline />);
    expect(input().tagName).toBe("TEXTAREA");
    type("first\nsecond\n");
    expect(screen.getByTestId("value").textContent).toBe(JSON.stringify("first\nsecond\n"));
  });
  test("creation starts empty with a reveal control", () => {
    const change = vi.fn();
    render(<SecretInput aria-label="Credential" isCreating value="" onValueChange={change} />);
    expect(input().placeholder).toBe("");
    type("new password");
    expect(change).toHaveBeenCalledWith("new password");
    expect(screen.getByRole("button", { name: "common.toggle-password-visibility" })).toBeTruthy();
  });
  test("disabled secrets cannot be cleared", () => {
    const change = vi.fn();
    render(<SecretInput aria-label="Credential" disabled value={undefined} onValueChange={change} />);
    fireEvent.click(screen.getByRole("button", { name: "common.clear" }));
    expect(change).not.toHaveBeenCalled();
  });
});
