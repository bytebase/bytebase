import { describe, expect, test, vi } from "vitest";
import {
  TIMESTAMP_COLUMN_MIN_WIDTH,
  TIMESTAMP_COLUMN_WIDTH,
} from "@/components/timestampColumn";
import { distributeColumnWidths } from "@/hooks/useColumnWidths";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

import { grantColumns } from "./ProjectAccessGrantsPage";

describe("grantColumns", () => {
  const columns = grantColumns((key) => key);
  const expiry = columns.findIndex((column) => column.key === "expiration");

  test.each([1100, 1300, 1500, 1800])(
    "opens the expiry whole, zone included, in a %ipx table",
    (containerWidth) => {
      expect(distributeColumnWidths(columns, containerWidth)[expiry]).toBe(
        TIMESTAMP_COLUMN_WIDTH.operational
      );
    }
  );

  test("lets a reader narrow the expiry to the date", () => {
    expect(columns[expiry].minWidth).toBe(TIMESTAMP_COLUMN_MIN_WIDTH);
  });
});
