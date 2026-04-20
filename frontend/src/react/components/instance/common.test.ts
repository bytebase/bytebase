import { describe, expect, test } from "vitest";
import { SSL_UPDATE_MASK_FIELDS } from "./tls";

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
