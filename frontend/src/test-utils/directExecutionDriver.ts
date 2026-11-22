import { act } from "@testing-library/react";

/**
 * Drives the direct-execution field inside a form test that stubs
 * `EnvironmentSelect` with `environmentSelectStub` and renders into the
 * document: the switch, and the stub's "pick Staging" target.
 */
export const directExecutionDriver = (flush: () => Promise<void>) => {
  const getSwitch = () =>
    document.querySelector("[role='switch']") as HTMLElement;
  return {
    getSwitch,
    async toggle(): Promise<void> {
      await act(async () => {
        getSwitch().click();
      });
      await flush();
    },
    async pickStaging(): Promise<void> {
      await act(async () => {
        (
          document.querySelector("[data-testid='pick-staging']") as HTMLElement
        ).click();
      });
      await flush();
    },
  };
};
