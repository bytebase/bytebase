import { render } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import { IssueStatusIcon } from "./IssueStatusIcon";

describe("IssueStatusIcon", () => {
  test("uses Lucide glyphs for completed and canceled issues", () => {
    const { container, rerender } = render(
      <IssueStatusIcon status={IssueStatus.DONE} />
    );
    expect(container.querySelector(".lucide-check")).not.toBeNull();

    rerender(<IssueStatusIcon status={IssueStatus.CANCELED} />);
    expect(container.querySelector(".lucide-minus")).not.toBeNull();
  });
});
