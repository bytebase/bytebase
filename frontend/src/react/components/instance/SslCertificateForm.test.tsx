import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { SslCertificateForm } from "./SslCertificateForm";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe("SslCertificateForm", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders the CA trust hint for inline PEM input", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          ca=""
          cert=""
          sslKey=""
          engineType={Engine.POSTGRES}
        />
      );
    });

    expect(container.textContent).toContain(
      "data-source.ssl.ca-empty-uses-system-trust"
    );

    act(() => {
      root.unmount();
    });
  });
});
