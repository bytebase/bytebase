import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter, useLocation, useNavigate } from "react-router";
import { describe, expect, test } from "vitest";
import {
  createAdvancedSearchParser,
  useURLSearchParam,
} from "./useURLSearchParam";

describe("createAdvancedSearchParser", () => {
  test("keeps unsupported scope-looking tokens as query text", () => {
    const parse = createAdvancedSearchParser(["state", "label"]);

    expect(parse("state:ALL project:foo bar")).toEqual({
      query: "bar project:foo",
      scopes: [{ id: "state", value: "ALL" }],
    });
  });
});

describe("useURLSearchParam", () => {
  test("uses the URL as the source of truth across Back and Forward", async () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <MemoryRouter
        initialEntries={["/items?q=open", "/items?q=closed"]}
        initialIndex={1}
      >
        {children}
      </MemoryRouter>
    );

    const { result } = renderHook(
      () => ({
        state: useURLSearchParam({ param: "q", defaultValue: "default" }),
        navigate: useNavigate(),
      }),
      { wrapper }
    );

    expect(result.current.state[0]).toBe("closed");
    await act(() => result.current.navigate(-1));
    expect(result.current.state[0]).toBe("open");
    await act(() => result.current.navigate(1));
    expect(result.current.state[0]).toBe("closed");
  });

  test("preserves unrelated parameters and explicit empty values", async () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <MemoryRouter initialEntries={["/items?panel=activity"]}>
        {children}
      </MemoryRouter>
    );

    const { result } = renderHook(
      () => ({
        state: useURLSearchParam({ param: "q", defaultValue: "default" }),
        location: useLocation(),
      }),
      { wrapper }
    );

    act(() => result.current.state[1](""));
    expect(result.current.location.search).toBe("?panel=activity&q=");
  });

  test("omits a value equal to the default from the URL", () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <MemoryRouter initialEntries={["/items?panel=activity&q=closed"]}>
        {children}
      </MemoryRouter>
    );

    const { result } = renderHook(
      () => ({
        state: useURLSearchParam({ param: "q", defaultValue: "default" }),
        location: useLocation(),
      }),
      { wrapper }
    );

    act(() => result.current.state[1]("default"));
    expect(result.current.location.search).toBe("?panel=activity");
  });

  test("skips navigation when the URL would not change", () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <MemoryRouter initialEntries={["/items?q=closed"]}>
        {children}
      </MemoryRouter>
    );

    const { result } = renderHook(
      () => ({
        state: useURLSearchParam({ param: "q", defaultValue: "default" }),
        location: useLocation(),
      }),
      { wrapper }
    );

    const locationBefore = result.current.location;
    act(() => result.current.state[1]("closed"));
    expect(result.current.location).toBe(locationBefore);
  });
});
