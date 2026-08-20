import i18next from "i18next";
import { describe, expect, test } from "vitest";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";

const LOCALES = {
  "en-US": enUS,
  "zh-CN": zhCN,
  "es-ES": esES,
  "ja-JP": jaJP,
  "vi-VN": viVN,
} as const;

describe("Instance form locale interpolation", () => {
  test.each(Object.entries(LOCALES))(
    "renders host and proxy placeholders literally in %s",
    (locale, translation) => {
      const i18n = i18next.createInstance();

      void i18n.init({
        resources: {
          [locale]: { translation },
        },
        lng: locale,
        interpolation: {
          escapeValue: false,
        },
        initAsync: false,
      });

      expect(i18n.t("instance.sentence.host.none-snowflake")).toContain(" | ");
      expect(i18n.t("instance.sentence.host.none-snowflake")).not.toContain(
        "{"
      );
      expect(i18n.exists("instance.sentence.host.saas")).toBe(true);
      expect(
        i18n.exists("instance.failed-to-connect-instance-saas-local-host")
      ).toBe(true);
      expect(
        i18n.exists(
          "instance.connection-recovery.network.description-self-hosted"
        )
      ).toBe(true);
      expect(
        i18n.exists("instance.connection-recovery.network.description-saas")
      ).toBe(true);
      expect(i18n.t("instance.sentence.proxy.snowflake")).toContain("@");
      expect(i18n.t("instance.sentence.proxy.snowflake")).not.toContain("{");
    }
  );

  test("uses concise English connection status messages", () => {
    expect(enUS.instance["failed-to-connect-instance"]).toBe(
      "Connection test failed."
    );
    expect(enUS.instance["successfully-connected-instance"]).toBe(
      "Connection successful."
    );
    expect(enUS.instance["unable-to-connect"]).toContain(
      "Review the guidance above, or continue"
    );
    expect(enUS.instance.sentence["firewall-info"]).toBe(
      "Allowlist the Bytebase Cloud IP in your database firewall."
    );
  });
});
