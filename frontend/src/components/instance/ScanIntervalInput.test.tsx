import { create } from "@bufbuild/protobuf";
import { type Duration, DurationSchema } from "@bufbuild/protobuf/wkt";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { ScanIntervalInput } from "./ScanIntervalInput";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/FeatureBadge", () => ({ FeatureBadge: () => null }));

afterEach(cleanup);

let latest: Duration | undefined;
function Harness({ seconds }: { seconds: number }) {
  const [interval, setInterval] = useState<Duration | undefined>(
    create(DurationSchema, { seconds: BigInt(seconds) })
  );
  latest = interval;
  return (
    <ScanIntervalInput
      allowEdit
      scanInterval={interval}
      onScanIntervalChange={setInterval}
    />
  );
}

const toggle = () =>
  screen.getByRole("switch", { name: "instance.scan-interval.self" });

test("an interval of zero is off, and turning it on starts from a day", () => {
  render(<Harness seconds={0} />);
  expect(toggle()).not.toBeChecked();
  expect(screen.queryByRole("spinbutton")).toBeNull();
  fireEvent.click(toggle());
  expect(Number(latest?.seconds)).toBe(24 * 60 * 60);
  expect(screen.getByRole("spinbutton")).toHaveValue(24 * 60);
});

test("keeps the last valid interval while the typed one is below the minimum", () => {
  render(<Harness seconds={3600} />);
  expect(toggle()).toBeChecked();
  fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "10" } });
  expect(screen.getByText("instance.scan-interval.min-value")).toBeVisible();
  expect(Number(latest?.seconds)).toBe(3600);
  fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "45" } });
  expect(Number(latest?.seconds)).toBe(45 * 60);
  fireEvent.click(toggle());
  expect(Number(latest?.seconds)).toBe(0);
});
