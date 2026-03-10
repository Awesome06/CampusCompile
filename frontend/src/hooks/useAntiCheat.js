import { useEffect, useRef, useCallback } from 'react';
import api from '../services/api';

export default function useAntiCheat(contestId, isContest) {
  // Refs to persist state without triggering re-renders
  const keyBuffer = useRef([]);
  const mouseLeaveTimer = useRef(null);

  // Helper to dispatch telemetry to the Go backend
  const dispatchTelemetry = useCallback((eventType, metadata = {}) => {
    if (!isContest || !contestId) return;
    
    // Fire and forget - don't await so we don't block the UI
    api.post(`/contests/${contestId}/telemetry`, {
      event_type: eventType,
      metadata: metadata
    }).catch(err => console.error("Telemetry dispatch failed", err));
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
      // If the mouse leaves the viewport, start a 45-second timer
      mouseLeaveTimer.current = setTimeout(() => {
        // If 45s pass and a 'blur' event hasn't fired natively, 
        // they are likely using a visibility-spoofing extension.
        if (document.hasFocus()) {
          dispatchTelemetry('visibility_spoof_suspected', { 
            duration_out: 45000 
          });
        }
      }, 45000);
    };

    const handleMouseEnter = () => {
      // Mouse came back, cancel the trap
      if (mouseLeaveTimer.current) {
        clearTimeout(mouseLeaveTimer.current);
      }
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
  
  // Call this manually when a paste is intercepted
  const logPasteAttempt = useCallback(() => {
    dispatchTelemetry('paste_attempt', { timestamp: Date.now() });
  }, [dispatchTelemetry]);

  // Call this on every keystroke in the Monaco editor
  const logKeystroke = useCallback(() => {
    if (!isContest) return;

    const now = Date.now();
    keyBuffer.current.push(now);

    // Keep the buffer at exactly 50 keystrokes to run the math
    if (keyBuffer.current.length > 50) {
      keyBuffer.current.shift();
    }

    if (keyBuffer.current.length === 50) {
      const deltas = [];
      for (let i = 1; i < keyBuffer.current.length; i++) {
        deltas.push(keyBuffer.current[i] - keyBuffer.current[i - 1]);
      }

      // Calculate Mean (Average delay between keys)
      const mean = deltas.reduce((a, b) => a + b, 0) / deltas.length;

      // Calculate Standard Deviation (Variance)
      const variance = deltas.reduce((a, b) => a + Math.pow(b - mean, 2), 0) / deltas.length;
      const stdDev = Math.sqrt(variance);

      // Math: 50 keystrokes = ~10 words. Calculate WPM.
      const timeFor50KeysMins = (now - keyBuffer.current[0]) / 60000;
      const wpm = 10 / timeFor50KeysMins;

      // The Trigger: Superhuman speed AND robotic consistency
      if (wpm > 180 && stdDev < 15) {
        dispatchTelemetry('autotyper_suspected', {
          wpm: Math.round(wpm),
          std_dev_ms: Math.round(stdDev),
          mean_delay_ms: Math.round(mean)
        });
        
        // Flush the buffer so we don't spam the server for every subsequent key
        keyBuffer.current = []; 
      }
    }
  }, [isContest, dispatchTelemetry]);

  return { logPasteAttempt, logKeystroke };
}