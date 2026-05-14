/**
 * useWebSocket — React hook for WebSocket connection lifecycle.
 *
 * Connects on mount, disconnects on unmount, auto-reconnects via the
 * underlying wsMiddleware. Exposes connection status and a manual reconnect.
 */

import { useCallback, useEffect, useState } from 'react';
import {
  startWs,
  stopWs,
  subscribeWsStatus,
  getWsStatus,
  type WsStatus,
} from '../store/wsMiddleware';

export type { WsStatus };

export interface UseWebSocketResult {
  status: WsStatus;
  reconnect: () => void;
}

export function useWebSocket(): UseWebSocketResult {
  const [status, setStatus] = useState<WsStatus>(getWsStatus);

  useEffect(() => {
    startWs();
    const unsubscribe = subscribeWsStatus(setStatus);
    return () => {
      unsubscribe();
      stopWs();
    };
  }, []);

  const reconnect = useCallback(() => {
    stopWs();
    startWs();
  }, []);

  return { status, reconnect };
}
