import { useEffect, useRef } from 'react';
import useAuthToken from './useAuthToken';

const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api';

export default function useHeartbeat(contestId, isElevated, hasEnteredArena) {
  const token = useAuthToken();
  const heartbeatIntervalRef = useRef(null);

  useEffect(() => {
    // Only start heartbeat for students who have entered the arena
    if (isElevated || !hasEnteredArena || !token || !contestId) {
      if (heartbeatIntervalRef.current) {
        clearInterval(heartbeatIntervalRef.current);
        heartbeatIntervalRef.current = null;
      }
      return;
    }

    const sendHeartbeat = async () => {
      try {
        await fetch(`${BASE_URL}/contests/${contestId}/heartbeat`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`
          }
        });
      } catch (err) {
        console.error('Failed to send heartbeat:', err);
      }
    };

    // Send immediately 
    sendHeartbeat();

    // Then send every 10 seconds
    heartbeatIntervalRef.current = setInterval(sendHeartbeat, 10000);

    return () => {
      if (heartbeatIntervalRef.current) {
        clearInterval(heartbeatIntervalRef.current);
        heartbeatIntervalRef.current = null;
      }
    };
  }, [contestId, isElevated, hasEnteredArena, token]);
}
