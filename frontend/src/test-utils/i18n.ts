/**
 * A react-i18next stand-in that keeps what a test might be about: the key,
 * and the variables interpolated into it (appended as JSON so an assertion
 * can read them back). `vi.mock` is hoisted above imports, so use it as
 * `vi.mock("react-i18next", async () =>
 *   (await import("@/test-utils/i18n")).reactI18nextStub())`.
 */
export const reactI18nextStub = () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key} ${JSON.stringify(vars)}` : key,
    i18n: { language: "en-US" },
  }),
});
