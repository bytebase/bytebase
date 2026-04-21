import { describe, expect, test } from "vitest";
import {
  applyLocalTlsCaSource,
  applyLocalTlsClientCertSource,
  getLocalTlsCaSource,
  getLocalTlsClientCertSource,
  SSL_UPDATE_MASK_FIELDS,
} from "./tls";

describe("TLS update mask fields", () => {
  test("includes the SSL path fields alongside inline material", () => {
    expect(SSL_UPDATE_MASK_FIELDS).toEqual([
      "use_ssl",
      "ssl_ca",
      "ssl_cert",
      "ssl_key",
      "ssl_ca_path",
      "ssl_cert_path",
      "ssl_key_path",
    ]);
  });
});

describe("TLS local source helpers", () => {
  test("treats empty CA material with SSL enabled as system trust", () => {
    expect(getLocalTlsCaSource({ useSsl: true })).toBe("SYSTEM_TRUST");
  });

  test("clears only CA fields when selecting system trust", () => {
    const next = applyLocalTlsCaSource(
      {
        useSsl: true,
        sslCa: "inline-ca",
        sslCaPath: "/tmp/ca.pem",
        sslCert: "inline-cert",
        sslKeyPath: "/tmp/key.pem",
        hasSslCertPath: true,
        hasSslKeyPath: true,
      } as never,
      "SYSTEM_TRUST"
    );
    expect(next.sslCa).toBe("");
    expect(next.sslCaPath).toBe("");
    expect(next.sslCert).toBe("inline-cert");
    expect(next.sslKeyPath).toBe("/tmp/key.pem");
    expect(next.hasSslCertPath).toBe(true);
    expect(next.hasSslKeyPath).toBe(true);
  });

  test("clears only client cert fields when selecting none", () => {
    const next = applyLocalTlsClientCertSource(
      {
        useSsl: true,
        sslCaPath: "/tmp/ca.pem",
        sslCert: "inline-cert",
        sslKey: "inline-key",
        sslCertPath: "/tmp/cert.pem",
        sslKeyPath: "/tmp/key.pem",
        hasSslCaPath: true,
      } as never,
      "NONE"
    );
    expect(next.sslCaPath).toBe("/tmp/ca.pem");
    expect(next.hasSslCaPath).toBe(true);
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
    expect(next.sslCertPath).toBe("");
    expect(next.sslKeyPath).toBe("");
  });

  test("infers client certificate source from path presence flags", () => {
    expect(
      getLocalTlsClientCertSource({
        useSsl: true,
        hasSslCertPath: true,
      } as never)
    ).toBe("FILE_PATH");
  });

  test("infers client certificate source from inline presence flags", () => {
    expect(
      getLocalTlsClientCertSource({
        useSsl: true,
        hasSslCert: true,
      } as never)
    ).toBe("INLINE_PEM");
  });

  test("infers CA source from inline presence flag", () => {
    expect(
      getLocalTlsCaSource({
        useSsl: true,
        hasSslCa: true,
      } as never)
    ).toBe("INLINE_PEM");
  });
});
