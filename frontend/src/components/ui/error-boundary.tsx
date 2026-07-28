import { Component, type ReactNode } from "react";

interface ErrorBoundaryProps {
  /** Rendered in place of children after a render error is caught. */
  readonly fallback: ReactNode;
  /** Called once per caught error — wire to console.error or telemetry. */
  readonly onError?: (error: unknown) => void;
  /**
   * When this value changes (by `Object.is`) after an error, the boundary
   * clears the error and re-attempts rendering children. Pass the data the
   * children render so fresh input gets a fresh chance.
   */
  readonly resetKey?: unknown;
  readonly children: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
}

/**
 * Containment for render-time exceptions: a crash inside `children` swaps in
 * `fallback` instead of unmounting the application. Use it around components
 * that render data they don't control (e.g. third-party tree/virtualization
 * libraries that throw on malformed input).
 */
export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  state: ErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true };
  }

  override componentDidCatch(error: unknown) {
    this.props.onError?.(error);
  }

  override componentDidUpdate(prevProps: ErrorBoundaryProps) {
    if (
      this.state.hasError &&
      !Object.is(prevProps.resetKey, this.props.resetKey)
    ) {
      this.setState({ hasError: false });
    }
  }

  override render() {
    if (this.state.hasError) {
      return this.props.fallback;
    }
    return this.props.children;
  }
}
