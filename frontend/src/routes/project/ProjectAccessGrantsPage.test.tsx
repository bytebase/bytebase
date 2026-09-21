import { describe, expect, test, vi } from "vitest";
import { TIMESTAMP_COLUMN } from "@/components/timestampColumn";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

import { grantColumns } from "./ProjectAccessGrantsPage";

describe("grantColumns", () => {
  test("keeps the expiry's zone in view at any width", () => {
    // An operational time carries its zone so a reader elsewhere knows when
    // access really ends, and the zone is the end of the string -- the first
    // thing a narrow column cuts. So the column is sized to the whole form,
    // and a drag stops there.
    const expiration = grantColumns((key) => key).find(
      (column) => column.key === "expiration"
    );
    expect(expiration?.defaultWidth).toBe(TIMESTAMP_COLUMN.operational.width);
    expect(expiration?.minWidth).toBe(TIMESTAMP_COLUMN.operational.width);
  });
});
