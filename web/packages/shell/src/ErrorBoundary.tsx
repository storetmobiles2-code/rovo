import { Component, type ErrorInfo, type ReactNode } from 'react';
import { ErrorState } from '@rovo/ui';

interface Props {
  children: ReactNode;
  /** Custom fallback; receives the error and a reset callback. */
  fallback?: (error: unknown, reset: () => void) => ReactNode;
  /** Error sink (Grafana Faro in production, doc 17 §13.3). */
  onError?: (error: unknown, info: ErrorInfo) => void;
}

interface State {
  error: unknown;
}

/**
 * Top-level React error boundary. A repeated chunk-load failure (after a deploy) should be
 * handled by the app's "Reload app" action, which can also unregister a broken SW (doc 18 §5).
 */
export class ErrorBoundary extends Component<Props, State> {
  override state: State = { error: null };

  static getDerivedStateFromError(error: unknown): State {
    return { error };
  }

  override componentDidCatch(error: unknown, info: ErrorInfo): void {
    this.props.onError?.(error, info);
    // eslint-disable-next-line no-console
    console.error('Uncaught error in React tree', error);
  }

  reset = (): void => this.setState({ error: null });

  override render(): ReactNode {
    if (this.state.error == null) return this.props.children;
    if (this.props.fallback) return this.props.fallback(this.state.error, this.reset);
    return (
      <div className="flex min-h-dvh items-center justify-center bg-bg">
        <ErrorState error={this.state.error} onRetry={() => window.location.reload()} />
      </div>
    );
  }
}
