import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { expect, test, vi } from "vitest";
import { SslCertificateForm } from "./SslCertificateForm";
import { ValidationField, ValidationInput, ValidationProvider } from "./ValidationField";

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));

test("shows a field error on blur, associates it with the input, and clears it when corrected", () => {
  const form = (errors: Record<string, string>) => (
    <ValidationProvider errors={errors}>
      <ValidationField validationField="host" title="Host">
        <ValidationInput defaultValue="" />
      </ValidationField>
    </ValidationProvider>
  );
  const view = render(form({ host: "required" }));
  const input = screen.getByRole("textbox", { name: "Host" });
  expect(screen.queryByText("instance.validation.required")).toBeNull();
  fireEvent.blur(input);
  const error = screen.getByText("instance.validation.required");
  expect(input.getAttribute("aria-invalid")).toBe("true");
  expect(input.getAttribute("aria-describedby")).toBe(error.id);
  view.rerender(form({}));
  expect(screen.queryByText("instance.validation.required")).toBeNull();
  expect(input.getAttribute("aria-invalid")).not.toBe("true");
  view.unmount();
});

test("TLS path feedback is visible before submitting and clears while correcting the field", () => {
  function Form() {
    const [path, setPath] = useState("certs/ca.pem");
    return (
      <ValidationProvider errors={path && !path.startsWith("/") ? { sslCaPath: "absolute-path" } : {}}>
        <SslCertificateForm useSsl verify caSource="FILE_PATH" onCaSourceChange={() => {}} caPath={path} onCaPathChange={setPath} />
      </ValidationProvider>
    );
  }
  const view = render(<Form />);
  const input = screen.getByTestId("tls-ca-path-input");
  expect(screen.queryByText("instance.validation.absolute-path")).toBeNull();
  fireEvent.blur(input);
  expect(screen.getByText("instance.validation.absolute-path")).toBeTruthy();
  expect(input.getAttribute("aria-invalid")).toBe("true");
  fireEvent.change(input, { target: { value: "/certs/ca.pem" } });
  expect(screen.queryByText("instance.validation.absolute-path")).toBeNull();
  view.unmount();
});
