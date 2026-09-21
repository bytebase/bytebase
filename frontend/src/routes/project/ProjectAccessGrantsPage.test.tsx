import { describe, expect, test, vi } from "vitest";
import { TIMESTAMP_COLUMN } from "@/components/timestampColumn";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

import { grantColumns } from "./ProjectAccessGrantsPage";

describe("grantColumns", () => {
  test("shows the expiry's zone by default, and lets the reader trade it", () => {
    // An operational time carries its zone so a reader elsewhere knows when
    // access really ends, and the zone is the end of the string -- the first
    // thing a narrow column cuts. So the column opens wide enough for all of
    // it; narrower is the reader's call, not a floor set by the widest zone.
    const expiration = grantColumns((key) => key).find(
      (column) => column.key === "expiration"
    );
    expect(expiration?.defaultWidth).toBe(TIMESTAMP_COLUMN.operational.width);
    expect(expiration?.minWidth).toBeLessThan(
      TIMESTAMP_COLUMN.operational.width
    );
  });
});
