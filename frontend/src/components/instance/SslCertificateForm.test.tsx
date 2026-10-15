import type { ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { fireEvent, screen } from "@testing-library/react";
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

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({
    content,
    children,
  }: {
    content: ReactNode;
    children: ReactNode;
  }) => (
    <span data-tooltip={typeof content === "string" ? content : undefined}>
      {children}
      {content}
    </span>
  ),
}));

describe("SslCertificateForm", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("does not render the CA trust hint", () => {
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

    expect(container.textContent).not.toContain(
      "data-source.ssl.ca-empty-uses-system-trust"
    );

    act(() => {
      root.unmount();
    });
  });

  test("renders the verification switch before its label in an inline row", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          layout="vertical"
          verifyControlLayout="inline"
          verify={true}
          onVerifyChange={() => {}}
        />
      );
    });

    const verifySwitch = screen.getByRole("switch", {
      name: "data-source.ssl.verify-certificate",
    });
    expect(verifySwitch.parentElement).toHaveAttribute(
      "data-slot",
      "form-control-row"
    );
    expect(verifySwitch.parentElement?.lastElementChild?.textContent).toBe(
      "data-source.ssl.verify-certificate"
    );

    act(() => {
      root.unmount();
    });
  });

  test("renders posture-first connection security controls", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          posture="TLS"
          onPostureChange={() => {}}
          caSource="SYSTEM_TRUST"
          onCaSourceChange={() => {}}
          clientCertSource="NONE"
          onClientCertSourceChange={() => {}}
          useSsl={true}
          verify={true}
          onVerifyChange={() => {}}
          engineType={Engine.POSTGRES}
        />
      );
    });

    expect(container.textContent).not.toContain(
      "data-source.ssl.connection-security"
    );
    expect(container.textContent).toContain("data-source.ssl.posture.tls");
    expect(container.textContent).not.toContain("data-source.ssl.posture.self");
    const posture = screen.getByRole("combobox", {
      name: "data-source.ssl.posture.self",
    });
    fireEvent.click(posture);
    expect(
      await screen.findByRole("option", {
        name: "data-source.ssl.posture.disabled",
      })
    ).toBeTruthy();
    expect(
      screen.getByRole("option", { name: "data-source.ssl.posture.tls" })
    ).toBeTruthy();
    expect(
      screen.getByRole("option", {
        name: "data-source.ssl.posture.mutual-tls",
      })
    ).toBeTruthy();
    expect(container.textContent).toContain("data-source.ssl.server-identity");
    expect(container.textContent).not.toContain(
      "data-source.ssl.ca-empty-uses-system-trust"
    );
    expect(container.textContent).not.toContain(
      "data-source.ssl.client-identity"
    );

    act(() => {
      root.unmount();
    });
  });

  test("renders client identity for mutual TLS without a None source option", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          posture="MUTUAL_TLS"
          onPostureChange={() => {}}
          caSource="SYSTEM_TRUST"
          onCaSourceChange={() => {}}
          clientCertSource="INLINE_PEM"
          onClientCertSourceChange={() => {}}
          useSsl={true}
          verify={true}
          onVerifyChange={() => {}}
          showKeyAndCert
          engineType={Engine.POSTGRES}
        />
      );
    });

    expect(container.textContent).toContain("data-source.ssl.client-identity");
    expect(container.textContent).toContain(
      "data-source.ssl.client-cert-source.inline-pem"
    );
    expect(container.textContent).not.toContain(
      "data-source.ssl.client-cert-source.none"
    );
    expect(
      Array.from(container.querySelectorAll("textarea")).map((textarea) =>
        textarea.getAttribute("placeholder")
      )
    ).toEqual([
      "data-source.ssl.client-cert-placeholder",
      "data-source.ssl.client-key-placeholder",
    ]);
    const clientCertSource = screen.getByRole("radio", {
      name: "data-source.ssl.client-cert-source.inline-pem",
    });
    expect(
      clientCertSource
    ).toHaveAttribute("aria-checked", "true");

    act(() => {
      root.unmount();
    });
  });

  test("hides CA controls when verification is disabled", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          posture="TLS"
          onPostureChange={() => {}}
          caSource="INLINE_PEM"
          onCaSourceChange={() => {}}
          clientCertSource="NONE"
          onClientCertSourceChange={() => {}}
          useSsl={true}
          verify={false}
          onVerifyChange={() => {}}
          engineType={Engine.POSTGRES}
        />
      );
    });

    expect(container.textContent).toContain(
      "data-source.ssl.verification-disabled-description"
    );
    expect(container.textContent).not.toContain(
      "data-source.ssl.ca-source.self"
    );
    expect(container.textContent).not.toContain(
      "data-source.ssl.ca-empty-uses-system-trust"
    );

    act(() => {
      root.unmount();
    });
  });

  test.each(["posture", "groups"])("keeps CA drafts editable after disabling verification in %s mode", (mode) => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const render = (verify: boolean) => act(() => {
      root.render(<SslCertificateForm
        posture={mode === "posture" ? "TLS" : undefined}
        onPostureChange={() => {}}
        caSource="FILE_PATH"
        onCaSourceChange={() => {}}
        clientCertSource="NONE"
        onClientCertSourceChange={() => {}}
        useSsl={true}
        verify={verify}
        onVerifyChange={() => {}}
        caPath="relative.pem"
        onCaPathChange={() => {}}
        engineType={Engine.POSTGRES}
      />);
    });
    try {
      render(true);
      expect(container.querySelector('input[value="relative.pem"]')).not.toBeNull();
      render(false);
      const input = container.querySelector<HTMLInputElement>('input[value="relative.pem"]');
      expect(input).not.toBeNull();
      expect(input?.disabled).toBe(false);
    } finally {
      act(() => root.unmount());
    }
  });

  test("falls back to legacy UI when posture source props are incomplete", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          posture="TLS"
          onPostureChange={() => {}}
          useSsl={true}
          onUseSslChange={() => {}}
          verify={true}
          onVerifyChange={() => {}}
          engineType={Engine.POSTGRES}
        />
      );
    });

    expect(container.textContent).not.toContain(
      "data-source.ssl.connection-security"
    );
    expect(container.textContent).toContain("data-source.ssl-connection");
    expect(container.textContent).not.toContain(
      "data-source.ssl.ca-empty-uses-system-trust"
    );

    act(() => {
      root.unmount();
    });
  });

  test("disables file path source options in SaaS mode", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          posture="MUTUAL_TLS"
          onPostureChange={() => {}}
          caSource="FILE_PATH"
          onCaSourceChange={() => {}}
          clientCertSource="FILE_PATH"
          onClientCertSourceChange={() => {}}
          useSsl={true}
          verify={true}
          onVerifyChange={() => {}}
          isSaaSMode
          showKeyAndCert
          engineType={Engine.POSTGRES}
        />
      );
    });

    fireEvent.click(
      screen.getByRole("combobox", {
        name: "data-source.ssl.ca-source.self",
      })
    );
    expect(
      await screen.findByRole("option", {
        name: "data-source.ssl.ca-source.file-path",
      })
    ).toHaveAttribute("aria-disabled", "true");
    expect(
      container.querySelector<HTMLInputElement>(
        '[data-testid="tls-ca-path-input"]'
      )?.disabled
    ).toBe(true);
    expect(
      container.querySelector<HTMLInputElement>(
        '[data-testid="tls-cert-path-input"]'
      )?.disabled
    ).toBe(true);
    expect(
      container.querySelector<HTMLInputElement>(
        '[data-testid="tls-key-path-input"]'
      )?.disabled
    ).toBe(true);
    act(() => {
      root.unmount();
    });
  });

  test("shows disabled mutual TLS for unsupported engines", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          posture="TLS"
          onPostureChange={() => {}}
          caSource="SYSTEM_TRUST"
          onCaSourceChange={() => {}}
          clientCertSource="NONE"
          onClientCertSourceChange={() => {}}
          useSsl={true}
          verify={true}
          onVerifyChange={() => {}}
          engineType={Engine.MSSQL}
        />
      );
    });

    fireEvent.click(
      screen.getByRole("combobox", {
        name: "data-source.ssl.posture.self",
      })
    );
    expect(
      await screen.findByRole("option", {
        name: "data-source.ssl.posture.mutual-tls",
      })
    ).toHaveAttribute("aria-disabled", "true");

    act(() => {
      root.unmount();
    });
  });

  test("falls back from mutual TLS for unsupported engines without saved client identity", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          posture="MUTUAL_TLS"
          onPostureChange={() => {}}
          caSource="SYSTEM_TRUST"
          onCaSourceChange={() => {}}
          clientCertSource="NONE"
          onClientCertSourceChange={() => {}}
          useSsl={true}
          verify={true}
          onVerifyChange={() => {}}
          engineType={Engine.MSSQL}
        />
      );
    });

    expect(
      screen.getByRole("combobox", {
        name: "data-source.ssl.posture.self",
      })
    ).toHaveTextContent("data-source.ssl.posture.tls");
    expect(container.textContent).not.toContain(
      "data-source.ssl.client-identity"
    );

    act(() => {
      root.unmount();
    });
  });

  test("does not treat non-none source as saved client identity for unsupported engines", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          posture="MUTUAL_TLS"
          onPostureChange={() => {}}
          caSource="SYSTEM_TRUST"
          onCaSourceChange={() => {}}
          clientCertSource="FILE_PATH"
          onClientCertSourceChange={() => {}}
          useSsl={true}
          verify={true}
          onVerifyChange={() => {}}
          engineType={Engine.MSSQL}
        />
      );
    });

    expect(
      screen.getByRole("combobox", {
        name: "data-source.ssl.posture.self",
      })
    ).toHaveTextContent("data-source.ssl.posture.tls");
    expect(container.textContent).not.toContain(
      "data-source.ssl.client-identity"
    );

    act(() => {
      root.unmount();
    });
  });

  test("renders saved mutual TLS identity for unsupported engines", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          posture="MUTUAL_TLS"
          onPostureChange={() => {}}
          caSource="SYSTEM_TRUST"
          onCaSourceChange={() => {}}
          clientCertSource="INLINE_PEM"
          onClientCertSourceChange={() => {}}
          useSsl={true}
          verify={true}
          onVerifyChange={() => {}}
          hasCert={true}
          hasKey={true}
          engineType={Engine.MSSQL}
        />
      );
    });

    expect(container.textContent).toContain("data-source.ssl.client-identity");
    expect(container.textContent).toContain("data-source.ssl.client-cert");
    expect(container.textContent).toContain("data-source.ssl.client-key");
    expect(
      screen.getByRole("combobox", {
        name: "data-source.ssl.posture.self",
      })
    ).toHaveTextContent("data-source.ssl.posture.mutual-tls");

    act(() => {
      root.unmount();
    });
  });

  test("marks write-only TLS material as configured", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SslCertificateForm
          useSsl={true}
          caSource="INLINE_PEM"
          onCaSourceChange={() => {}}
          clientCertSource="FILE_PATH"
          onClientCertSourceChange={() => {}}
          hasCa={true}
          hasCertPath={true}
          hasKeyPath={true}
          showKeyAndCert={true}
          verify={true}
        />
      );
    });

    expect(
      container.querySelectorAll('[data-testid="tls-configured-badge"]')
    ).toHaveLength(3);
    expect(container.textContent).toContain("data-source.ssl.configured");

    act(() => {
      root.unmount();
    });
  });
});
