import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, test } from "vitest";
import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "./alert-dialog";
import { getLayerRoot, LAYER_ROOT_ID } from "./layer";

afterEach(() => {
  cleanup();
  document.body.innerHTML = "";
});

function Confirmation({ layer }: { layer?: "overlay" | "agent" }) {
  return (
    <AlertDialog>
      <AlertDialogTrigger>Review SQL</AlertDialogTrigger>
      <AlertDialogContent layer={layer}>
        <AlertDialogTitle>Existing SQL</AlertDialogTitle>
        <AlertDialogDescription>Review before replacing SQL.</AlertDialogDescription>
        <AlertDialogFooter>
          <AlertDialogClose>Cancel</AlertDialogClose>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

describe("AlertDialog", () => {
  test.each([undefined, "overlay", "agent"] as const)(
    "uses the %s layer without changing dialog behavior or styling",
    async (layer) => {
      render(<Confirmation layer={layer} />);
      const trigger = screen.getByRole("button", { name: "Review SQL" });
      fireEvent.click(trigger);
      const dialog = await screen.findByRole("alertdialog", { name: "Existing SQL" });
      expect(dialog.closest("[data-bb-layer-family]")?.id).toBe(LAYER_ROOT_ID[layer ?? "overlay"]);
      expect(dialog).toHaveAccessibleDescription("Review before replacing SQL.");
      expect(dialog).toHaveClass("p-6", "max-w-md", "rounded-sm");
      expect(dialog).not.toHaveAttribute("layer");

      fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
      await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
      await waitFor(() => expect(trigger).toHaveFocus());
    }
  );

  test.each([
    { dialogLayer: "overlay", higherLayer: "agent" },
    { dialogLayer: "agent", higherLayer: "critical" },
  ] as const)("preserves access to $higherLayer above a $dialogLayer confirmation", async ({ dialogLayer, higherLayer }) => {
    const higherRoot = getLayerRoot(higherLayer);
    const action = document.createElement("button");
    action.textContent = "Higher-layer action";
    higherRoot.appendChild(action);
    render(<Confirmation layer={dialogLayer} />);
    fireEvent.click(screen.getByRole("button", { name: "Review SQL" }));
    await screen.findByRole("alertdialog", { name: "Existing SQL" });

    await waitFor(() => {
      expect(higherRoot).not.toHaveAttribute("aria-hidden");
      expect(higherRoot).not.toHaveAttribute("inert");
      expect(higherRoot).not.toHaveAttribute("data-base-ui-inert");
    });
  });
});
