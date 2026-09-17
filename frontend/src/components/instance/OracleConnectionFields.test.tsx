import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { OracleConnectionFields } from "./OracleConnectionFields";
import { ValidationProvider } from "./ValidationField";

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
afterEach(cleanup);

function Form({ initial = { sid: "", serviceName: "" }, onChange = vi.fn(), allowEdit = true }) {
  const [identifier, setIdentifier] = useState(initial);
  const [resetEvent, setResetEvent] = useState(0);
  return (
    <ValidationProvider errors={!identifier.sid && !identifier.serviceName ? { serviceName: "oracle-service" } : {}}>
      <OracleConnectionFields dataSourceId="admin" instanceName="instances/one" {...identifier} resetEvent={resetEvent} allowEdit={allowEdit} onChange={(next) => { setIdentifier(next); onChange(next); }} />
      <button type="button" onClick={() => { setIdentifier(initial); setResetEvent((event) => event + 1); }}>Revert</button>
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


test.each([
  { initial: { sid: "", serviceName: "sales.example.com" }, active: "instance.service-name", inactive: "instance.sid" },
  { initial: { sid: "ORCL", serviceName: "" }, active: "instance.sid", inactive: "instance.service-name" },
])("Revert clears hidden drafts when the saved $active value is already active", ({ initial, active, inactive }) => {
  render(<Form initial={initial} />);
  fireEvent.click(screen.getByRole("radio", { name: inactive }));
  fireEvent.change(screen.getByRole("textbox", { name: inactive }), { target: { value: "discarded-draft" } });
  fireEvent.click(screen.getByRole("radio", { name: active }));
  fireEvent.click(screen.getByRole("button", { name: "Revert" }));
  fireEvent.click(screen.getByRole("radio", { name: inactive }));
  expect(screen.getByRole("textbox", { name: inactive }).getAttribute("value")).toBe("");
});


function TabbedForm() {
  const saved = { admin: { sid: "", serviceName: "sales" }, readonly: { sid: "", serviceName: "sales" } };
  const [sources, setSources] = useState(saved);
  const [activeId, setActiveId] = useState<"admin" | "readonly">("admin");
  const [resetEvent, setResetEvent] = useState(0);
  const [instanceName, setInstanceName] = useState("instances/one");
  return <>
    <button type="button" onClick={() => setActiveId(activeId === "admin" ? "readonly" : "admin")}>Switch data source</button>
    <button type="button" onClick={() => { setSources(saved); setResetEvent((event) => event + 1); }}>Revert</button>
    <button type="button" onClick={() => { setSources(saved); setInstanceName("instances/two"); }}>Switch instance</button>
    <OracleConnectionFields dataSourceId={activeId} instanceName={instanceName} {...sources[activeId]} resetEvent={resetEvent} allowEdit onChange={(next) => setSources({ ...sources, [activeId]: next })} />
  </>;
}

test("keeps independent identifier drafts for data sources with identical saved values", () => {
  render(<TabbedForm />);
  fireEvent.click(screen.getByRole("radio", { name: "instance.sid" }));
  fireEvent.change(screen.getByRole("textbox", { name: "instance.sid" }), { target: { value: "ADMIN" } });
  fireEvent.click(screen.getByRole("radio", { name: "instance.service-name" }));
  fireEvent.click(screen.getByRole("button", { name: "Switch data source" }));
  fireEvent.click(screen.getByRole("radio", { name: "instance.sid" }));
  expect(screen.getByRole("textbox", { name: "instance.sid" }).getAttribute("value")).toBe("");
  fireEvent.change(screen.getByRole("textbox", { name: "instance.sid" }), { target: { value: "READONLY" } });
  fireEvent.click(screen.getByRole("button", { name: "Switch data source" }));
  expect(screen.getByRole("radio", { name: "instance.service-name" }).getAttribute("aria-checked")).toBe("true");
  fireEvent.click(screen.getByRole("radio", { name: "instance.sid" }));
  expect(screen.getByRole("textbox", { name: "instance.sid" }).getAttribute("value")).toBe("ADMIN");
  fireEvent.click(screen.getByRole("button", { name: "Switch data source" }));
  expect(screen.getByRole("textbox", { name: "instance.sid" }).getAttribute("value")).toBe("READONLY");
});

test.each(["Revert", "Switch instance"])("%s discards drafts for inactive data sources too", (action) => {
  render(<TabbedForm />);
  fireEvent.click(screen.getByRole("radio", { name: "instance.sid" }));
  fireEvent.change(screen.getByRole("textbox", { name: "instance.sid" }), { target: { value: "discarded" } });
  fireEvent.click(screen.getByRole("radio", { name: "instance.service-name" }));
  fireEvent.click(screen.getByRole("button", { name: "Switch data source" }));
  fireEvent.click(screen.getByRole("button", { name: action }));
  fireEvent.click(screen.getByRole("button", { name: "Switch data source" }));
  fireEvent.click(screen.getByRole("radio", { name: "instance.sid" }));
  expect(screen.getByRole("textbox", { name: "instance.sid" }).getAttribute("value")).toBe("");
});
