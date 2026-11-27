import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useLayoutEffect,
  useState,
} from "react";

const Register = createContext<(() => () => void) | undefined>(undefined);
const HasToolbarEntry = createContext(false);

export function AgentEntryPoints({ children }: { children: ReactNode }) {
  const [count, setCount] = useState(0);
  const register = useCallback(() => {
    setCount((value) => value + 1);
    return () => setCount((value) => value - 1);
  }, []);
  return (
    <Register.Provider value={register}>
      <HasToolbarEntry.Provider value={count > 0}>
        {children}
      </HasToolbarEntry.Provider>
    </Register.Provider>
  );
}

export function useToolbarEntry(fallback: boolean) {
  const register = useContext(Register);
  const hasToolbarEntry = useContext(HasToolbarEntry);
  useLayoutEffect(() => {
    if (!fallback) return register?.();
  }, [fallback, register]);
  return !fallback || !hasToolbarEntry;
}
