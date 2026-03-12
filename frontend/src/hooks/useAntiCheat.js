import { useEffect, useRef, useCallback } from 'react';
import api from '../services/api';

export default function useAntiCheat(contestId, isContest) {
  const keyBuffer = useRef([]);
  const mouseLeaveTimer = useRef(null);
  
  // 1. Create a buffer to hold events
  const eventBuffer = useRef([]);

  // 2. Modify dispatch to push to the buffer instead of calling the API directly
  const dispatchTelemetry = useCallback((eventType, metadata = {}) => {
    if (!isContest || !contestId) return;
    
    eventBuffer.current.push({
      event_type: eventType,
      metadata: metadata,
      timestamp: Date.now()
    });
  }, [contestId, isContest]);

  // 3. The Master Batching Daemon
  useEffect(() => {
    if (!isContest || !contestId) return;

    const flushTelemetry = () => {
      if (eventBuffer.current.length > 0) {
        // Send the entire array at once to the new batch endpoint
        api.post(`/contests/${contestId}/telemetry/batch`, {
          events: eventBuffer.current
        }).catch(err => console.error("Telemetry sync failed", err));
        
        // Clear the buffer
        eventBuffer.current = [];
      }
    };

    // Flush every 5 seconds
    const flushInterval = setInterval(flushTelemetry, 5000);

    // Attempt to flush if the user closes the tab or navigates away
    window.addEventListener('beforeunload', flushTelemetry);

    return () => {
      clearInterval(flushInterval);
      window.removeEventListener('beforeunload', flushTelemetry);
      flushTelemetry(); // Flush one last time on unmount
    };
  }, [contestId, isContest]);

  // ----------------------------------------------------
  // TRAP 1: Standard Window Blur (Tab Switching)
  // ----------------------------------------------------
  useEffect(() => {
    if (!isContest) return;

    const handleBlur = () => {
      dispatchTelemetry('blur', { timestamp: Date.now() });
    };

    window.addEventListener('blur', handleBlur);
    return () => window.removeEventListener('blur', handleBlur);
  }, [isContest, dispatchTelemetry]);

  // ----------------------------------------------------
  // TRAP 2: Always-Active Extensions (Mouse Leave)
  // ----------------------------------------------------
  useEffect(() => {
    if (!isContest) return;

    const handleMouseLeave = () => {
      mouseLeaveTimer.current = setTimeout(() => {
        if (document.hasFocus()) {
          dispatchTelemetry('visibility_spoof_suspected', { duration_out: 45000 });
        }
      }, 45000);
    };

    const handleMouseEnter = () => {
      if (mouseLeaveTimer.current) clearTimeout(mouseLeaveTimer.current);
    };

    document.addEventListener('mouseleave', handleMouseLeave);
    document.addEventListener('mouseenter', handleMouseEnter);

    return () => {
      document.removeEventListener('mouseleave', handleMouseLeave);
      document.removeEventListener('mouseenter', handleMouseEnter);
      if (mouseLeaveTimer.current) clearTimeout(mouseLeaveTimer.current);
    };
  }, [isContest, dispatchTelemetry]);

  // ----------------------------------------------------
  // TRAP 3 & 4: AutoTyper Detection & Paste Attempts
  // ----------------------------------------------------
  const logPasteAttempt = useCallback(() => {
    dispatchTelemetry('paste_attempt', { timestamp: Date.now() });
  }, [dispatchTelemetry]);

  const logKeystroke = useCallback(() => {
    if (!isContest) return;

    const now = Date.now();
    keyBuffer.current.push(now);

    if (keyBuffer.current.length > 50) keyBuffer.current.shift();

    if (keyBuffer.current.length === 50) {
      const deltas = [];
      for (let i = 1; i < keyBuffer.current.length; i++) {
        deltas.push(keyBuffer.current[i] - keyBuffer.current[i - 1]);
      }

      const mean = deltas.reduce((a, b) => a + b, 0) / deltas.length;
      const variance = deltas.reduce((a, b) => a + Math.pow(b - mean, 2), 0) / deltas.length;
      const stdDev = Math.sqrt(variance);

      const timeFor50KeysMins = (now - keyBuffer.current[0]) / 60000;
      const wpm = 10 / timeFor50KeysMins;

      if (wpm > 180 && stdDev < 15) {
        dispatchTelemetry('autotyper_suspected', {
          wpm: Math.round(wpm),
          std_dev_ms: Math.round(stdDev),
          mean_delay_ms: Math.round(mean)
        });
        keyBuffer.current = []; 
      }
    }
  }, [isContest, dispatchTelemetry]);

  return { logPasteAttempt, logKeystroke };
}