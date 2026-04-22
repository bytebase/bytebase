import type { ReactNode } from "react";
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

vi.mock("@/react/components/ui/tooltip", () => ({
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

  test("renders posture-first connection security controls", () => {
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
    expect(container.textContent).toContain("data-source.ssl.posture.disabled");
    expect(container.textContent).toContain("data-source.ssl.posture.tls");
    expect(container.textContent).toContain(
      "data-source.ssl.posture.mutual-tls"
    );
    expect(container.textContent).not.toContain("data-source.ssl.posture.self");
    expect(
      container.querySelector('[aria-label="data-source.ssl.posture.self"]')
    ).not.toBeNull();
    expect(
      container
        .querySelector('[aria-label="data-source.ssl.posture.self"]')
        ?.classList.contains("self-start")
    ).toBe(true);
    const selectedPostureInput = container.querySelector(
      '[aria-label="data-source.ssl.posture.self"] [aria-checked="true"]'
    );
    const selectedPostureLabel = selectedPostureInput?.closest("label");
    expect(
      Array.from(selectedPostureLabel?.classList ?? []).some((className) =>
        /^z-\d+$/.test(className)
      )
    ).toBe(false);
    expect(
      selectedPostureLabel?.classList.contains("focus-within:ring-inset")
    ).toBe(true);
    expect(
      selectedPostureLabel?.nextElementSibling?.classList.contains("border-l")
    ).toBe(false);
    expect(container.textContent).toContain("data-source.ssl.server-identity");
    expect(container.textContent).toContain(
      "data-source.ssl.ca-empty-uses-system-trust"
    );
    expect(container.textContent).not.toContain(
      "data-source.ssl.client-identity"
    );

    act(() => {
      root.unmount();
    });
  });

  test("renders client identity for mutual TLS without a None source option", () => {
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
    expect(container.textContent).toContain(
      "data-source.ssl.client-cert-source.file-path"
    );
    expect(container.textContent).not.toContain(
      "data-source.ssl.client-cert-source.none"
    );

    act(() => {
      root.unmount();
    });
  });

  test("keeps CA controls visible when verification is disabled", () => {
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
    expect(container.textContent).toContain("data-source.ssl.ca-source.self");
    expect(container.textContent).not.toContain(
      "data-source.ssl.ca-empty-uses-system-trust"
    );

    act(() => {
      root.unmount();
    });
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
    expect(container.textContent).toContain(
      "data-source.ssl.ca-empty-uses-system-trust"
    );

    act(() => {
      root.unmount();
    });
  });

  test("disables file path source options in SaaS mode", () => {
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

    expect(
      container.querySelector(
        '[aria-label="data-source.ssl.ca-source.self"] [aria-disabled="true"][aria-checked="true"]'
      )
    ).not.toBeNull();
    expect(
      container.querySelector(
        '[aria-label="data-source.ssl.client-cert-source.self"] [aria-disabled="true"][aria-checked="true"]'
      )
    ).not.toBeNull();
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
    expect(container.textContent).toContain(
      "data-source.ssl.ca-source.file-path-unavailable-saas"
    );

    act(() => {
      root.unmount();
    });
  });

  test("shows disabled mutual TLS for unsupported engines", () => {
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

    expect(container.textContent).toContain(
      "data-source.ssl.mutual-tls-unavailable-engine"
    );
    expect(container.querySelector('[aria-disabled="true"]')).not.toBeNull();

    act(() => {
      root.unmount();
    });
  });

  test("falls back from mutual TLS for unsupported engines without saved client identity", () => {
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

    expect(container.textContent).toContain(
      "data-source.ssl.mutual-tls-unavailable-engine"
    );
    expect(container.querySelector('[aria-disabled="true"]')).not.toBeNull();
    expect(
      container.querySelector(
        '[aria-label="data-source.ssl.posture.self"] [aria-disabled="true"][aria-checked="true"]'
      )
    ).toBeNull();
    expect(container.textContent).not.toContain(
      "data-source.ssl.client-identity"
    );

    act(() => {
      root.unmount();
    });
  });

  test("does not treat non-none source as saved client identity for unsupported engines", () => {
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

    expect(container.textContent).toContain(
      "data-source.ssl.mutual-tls-unavailable-engine"
    );
    expect(
      container.querySelector(
        '[aria-label="data-source.ssl.posture.self"] [aria-disabled="true"][aria-checked="true"]'
      )
    ).toBeNull();
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
      container.querySelector(
        '[aria-label="data-source.ssl.posture.self"] [aria-checked="true"][aria-disabled="true"]'
      )
    ).not.toBeNull();

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
