import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  applyLocalTlsCaSource,
  applyLocalTlsClientCertSource,
  applyLocalTlsPosture,
  getLocalTlsCaSource,
  getLocalTlsClientCertSource,
  getLocalTlsPosture,
  isLocalTlsClientIdentitySupported,
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
        sslCertPathSet: true,
        sslKeyPathSet: true,
      } as never,
      "SYSTEM_TRUST"
    );
    expect(next.sslCa).toBe("");
    expect(next.sslCaPath).toBe("");
    expect(next.sslCert).toBe("inline-cert");
    expect(next.sslKeyPath).toBe("/tmp/key.pem");
    expect(next.sslCertPathSet).toBe(true);
    expect(next.sslKeyPathSet).toBe(true);
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
        sslCaPathSet: true,
      } as never,
      "NONE"
    );
    expect(next.sslCaPath).toBe("/tmp/ca.pem");
    expect(next.sslCaPathSet).toBe(true);
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
    expect(next.sslCertPath).toBe("");
    expect(next.sslKeyPath).toBe("");
  });

  test("infers client certificate source from path presence flags", () => {
    expect(
      getLocalTlsClientCertSource({
        useSsl: true,
        sslCertPathSet: true,
      } as never)
    ).toBe("FILE_PATH");
  });

  test("infers client certificate source from inline presence flags", () => {
    expect(
      getLocalTlsClientCertSource({
        useSsl: true,
        sslCertSet: true,
      } as never)
    ).toBe("INLINE_PEM");
  });

  test("infers CA source from inline presence flag", () => {
    expect(
      getLocalTlsCaSource({
        useSsl: true,
        sslCaSet: true,
      } as never)
    ).toBe("INLINE_PEM");
  });
});

describe("TLS posture helpers", () => {
  test("infers disabled posture when SSL is off", () => {
    expect(getLocalTlsPosture({ useSsl: false })).toBe("DISABLED");
  });

  test("infers TLS posture when SSL is on without client identity", () => {
    expect(
      getLocalTlsPosture({
        useSsl: true,
        sslCaPathSet: true,
      } as never)
    ).toBe("TLS");
  });

  test("infers mutual TLS posture from inline client material", () => {
    expect(
      getLocalTlsPosture({
        useSsl: true,
        sslCertSet: true,
        sslKeySet: true,
      } as never)
    ).toBe("MUTUAL_TLS");
  });

  test("infers mutual TLS posture from file path client material", () => {
    expect(
      getLocalTlsPosture({
        useSsl: true,
        sslCertPathSet: true,
        sslKeyPathSet: true,
      } as never)
    ).toBe("MUTUAL_TLS");
  });

  test("switching posture to TLS clears only client identity fields", () => {
    const next = applyLocalTlsPosture(
      {
        useSsl: true,
        sslCaPath: "/tmp/ca.pem",
        sslCaPathSet: true,
        sslCert: "inline-cert",
        sslKey: "inline-key",
        sslCertPath: "/tmp/cert.pem",
        sslKeyPath: "/tmp/key.pem",
        sslCertSet: true,
        sslKeySet: true,
        sslCertPathSet: true,
        sslKeyPathSet: true,
      } as never,
      "TLS"
    );

    expect(next.useSsl).toBe(true);
    expect(next.sslCaPath).toBe("/tmp/ca.pem");
    expect(next.sslCaPathSet).toBe(true);
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
    expect(next.sslCertPath).toBe("");
    expect(next.sslKeyPath).toBe("");
    expect(next.sslCertSet).toBe(false);
    expect(next.sslKeySet).toBe(false);
    expect(next.sslCertPathSet).toBe(false);
    expect(next.sslKeyPathSet).toBe(false);
  });

  test("switching posture to disabled clears all TLS material", () => {
    const next = applyLocalTlsPosture(
      {
        useSsl: true,
        sslCa: "inline-ca",
        sslCaPath: "/tmp/ca.pem",
        sslCert: "inline-cert",
        sslKey: "inline-key",
      } as never,
      "DISABLED"
    );

    expect(next.useSsl).toBe(false);
    expect(next.sslCa).toBe("");
    expect(next.sslCaPath).toBe("");
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
  });

  test("MSSQL does not support client identity in this form", () => {
    expect(isLocalTlsClientIdentitySupported(Engine.MSSQL)).toBe(false);
    expect(isLocalTlsClientIdentitySupported(Engine.POSTGRES)).toBe(true);
  });
});
