import { describe, expect, test } from "vitest";
import { recoveryActionKinds } from "./connection-recovery-actions";

const base = {
  message: "",
  isSaaSMode: true,
  canUseSshTunnel: true,
  hasSampleInstance: true,
  verifiesServerCertificate: false,
};

describe("recoveryActionKinds", () => {
  test.each([
    "private_address",
    "host_not_found",
    "network_unreachable",
    "timeout",
  ] as const)(
    "offers the tunnel and the sample on Cloud for %s",
    (category) => {
      expect(recoveryActionKinds({ ...base, category })).toEqual([
        "ssh-tunnel",
        "sample",
      ]);
    }
  );

  test("offers neither when self-hosted", () => {
    expect(
      recoveryActionKinds({
        ...base,
        category: "network_unreachable",
        isSaaSMode: false,
      })
    ).toEqual([]);
  });

  test("leaves the tunnel out when the data source cannot use one", () => {
    expect(
      recoveryActionKinds({
        ...base,
        category: "timeout",
        canUseSshTunnel: false,
      })
    ).toEqual(["sample"]);
  });

  test("leaves the sample out when there is none to use", () => {
    expect(
      recoveryActionKinds({
        ...base,
        category: "private_address",
        hasSampleInstance: false,
      })
    ).toEqual(["ssh-tunnel"]);
  });

  test.each([
    "auth_failed",
    "permission_denied",
    "unsupported_engine",
    "unknown",
  ] as const)("offers nothing for %s", (category) => {
    expect(recoveryActionKinds({ ...base, category })).toEqual([]);
  });

  test("offers an unverified retry for a certificate Bytebase refused", () => {
    expect(
      recoveryActionKinds({
        ...base,
        category: "ssl_tls_failed",
        message:
          "tls: failed to verify certificate: x509: certificate signed by unknown authority",
        verifiesServerCertificate: true,
      })
    ).toEqual(["skip-tls-verify"]);
  });

  test("offers no unverified retry when verification is already off", () => {
    expect(
      recoveryActionKinds({
        ...base,
        category: "ssl_tls_failed",
        message: "x509: certificate has expired",
        verifiesServerCertificate: false,
      })
    ).toEqual([]);
  });

  test("offers no unverified retry when the server does not speak TLS", () => {
    expect(
      recoveryActionKinds({
        ...base,
        category: "ssl_tls_failed",
        message: "pq: SSL is not enabled on the server",
        verifiesServerCertificate: true,
      })
    ).toEqual([]);
  });
});
