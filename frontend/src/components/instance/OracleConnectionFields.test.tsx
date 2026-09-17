import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { OracleConnectionFields } from "./OracleConnectionFields";
import { ValidationProvider } from "./ValidationField";

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
afterEach(cleanup);

function Form({ initial = { sid: "", serviceName: "" }, onChange = vi.fn(), allowEdit = true }) {
  const [identifier, setIdentifier] = useState(initial);
  return (
    <ValidationProvider errors={!identifier.sid && !identifier.serviceName ? { serviceName: "oracle-service" } : {}}>
      <OracleConnectionFields {...identifier} allowEdit={allowEdit} onChange={(next) => { setIdentifier(next); onChange(next); }} />
      <button type="button" onClick={() => setIdentifier(initial)}>Revert</button>
    </ValidationProvider>
  );
}

test("defaults to an empty service name and keeps empty SID selected", () => {
  render(<Form />);
  expect(screen.getByRole("textbox", { name: "instance.service-name" }).getAttribute("value")).toBe("");
  fireEvent.click(screen.getByRole("radio", { name: "instance.sid" }));
  const input = screen.getByRole("textbox", { name: "instance.sid" });
  expect(input.getAttribute("value")).toBe("");
  fireEvent.change(input, { target: { value: "ORCL" } });
  fireEvent.change(input, { target: { value: "" } });
  expect(screen.getByRole("radio", { name: "instance.sid" }).getAttribute("aria-checked")).toBe("true");
  fireEvent.blur(input);
  expect(input.getAttribute("aria-invalid")).toBe("true");
  expect(input.getAttribute("aria-describedby")).toBe(screen.getByText("instance.validation.oracle-service").id);
});

test("preserves separate drafts and emits only the selected identifier in one update", () => {
  const onChange = vi.fn();
  render(<Form initial={{ sid: "", serviceName: "sales.example.com" }} onChange={onChange} />);
  fireEvent.click(screen.getByRole("radio", { name: "instance.sid" }));
  expect(onChange).toHaveBeenCalledTimes(1);
  expect(onChange).toHaveBeenLastCalledWith({ sid: "", serviceName: "" });
  fireEvent.change(screen.getByRole("textbox", { name: "instance.sid" }), { target: { value: "ORCL" } });
  fireEvent.click(screen.getByRole("radio", { name: "instance.service-name" }));
  expect(onChange).toHaveBeenLastCalledWith({ sid: "", serviceName: "sales.example.com" });
  expect(screen.getByRole("textbox", { name: "instance.service-name" }).getAttribute("value")).toBe("sales.example.com");
  fireEvent.click(screen.getByRole("radio", { name: "instance.sid" }));
  expect(onChange).toHaveBeenLastCalledWith({ sid: "ORCL", serviceName: "" });
  expect(screen.getByRole("textbox", { name: "instance.sid" }).getAttribute("value")).toBe("ORCL");
});

test("loads a saved SID and resets mode and drafts when reverted", () => {
  render(<Form initial={{ sid: "ORCL", serviceName: "" }} />);
  expect(screen.getByRole("textbox", { name: "instance.sid" }).getAttribute("value")).toBe("ORCL");
  fireEvent.click(screen.getByRole("radio", { name: "instance.service-name" }));
  fireEvent.change(screen.getByRole("textbox", { name: "instance.service-name" }), { target: { value: "draft.example.com" } });
  fireEvent.click(screen.getByRole("button", { name: "Revert" }));
  expect(screen.getByRole("textbox", { name: "instance.sid" }).getAttribute("value")).toBe("ORCL");
  fireEvent.click(screen.getByRole("radio", { name: "instance.service-name" }));
  expect(screen.getByRole("textbox", { name: "instance.service-name" }).getAttribute("value")).toBe("");
});

test("disables the identifier and mode choices without edit permission", () => {
  render(<Form allowEdit={false} />);
  expect((screen.getByRole("textbox") as HTMLInputElement).disabled).toBe(true);
  for (const radio of screen.getAllByRole("radio")) {
    expect(radio.getAttribute("aria-disabled")).toBe("true");
  }
});
