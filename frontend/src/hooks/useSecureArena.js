import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import useAntiCheat from './useAntiCheat';

export default function useSecureArena(contestId, hasStarted) {
  const { currentUser } = useAuth();
  
  // Securely gate elevated permissions using decoded JWT claim from AuthContext
  const isElevated = currentUser?.role === 'admin' || currentUser?.role === 'professor';

  const [isFullscreen, setIsFullscreen] = useState(false);
  const [tabViolations, setTabViolations] = useState(0);
  
  const { dispatchTelemetry } = useAntiCheat(contestId, !isElevated); // Initialize the daemon

  const toggleFullscreen = async () => {
    if (!document.fullscreenElement && !document.webkitFullscreenElement && !document.mozFullScreenElement && !document.msFullscreenElement) {
      try {
        const docElm = document.documentElement;
        if (docElm.requestFullscreen) await docElm.requestFullscreen();
        else if (docElm.webkitRequestFullscreen) await docElm.webkitRequestFullscreen();
        else if (docElm.mozRequestFullScreen) await docElm.mozRequestFullScreen();
        else if (docElm.msRequestFullscreen) await docElm.msRequestFullscreen();
      } catch (err) {
        console.error("Failed to enter fullscreen:", err);
      }
    } else {
      if (document.exitFullscreen) await document.exitFullscreen();
      else if (document.webkitExitFullscreen) await document.webkitExitFullscreen();
      else if (document.mozCancelFullScreen) await document.mozCancelFullScreen();
      else if (document.msExitFullscreen) await document.msExitFullscreen();
    }
  };

  useEffect(() => {
    let interval;

    const handleFullscreenChange = () => {
      const isCurrentlyFullscreen = !!(document.fullscreenElement || document.webkitFullscreenElement || document.mozFullScreenElement || document.msFullscreenElement);
      setIsFullscreen(isCurrentlyFullscreen);

      if (!isCurrentlyFullscreen && !isElevated && hasStarted) {
        dispatchTelemetry("fullscreen_dropped", { action: "exited_early" });
      }
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    document.addEventListener('webkitfullscreenchange', handleFullscreenChange);
    document.addEventListener('mozfullscreenchange', handleFullscreenChange);
    document.addEventListener('MSFullscreenChange', handleFullscreenChange);

    const initialFullscreen = !!(document.fullscreenElement || document.webkitFullscreenElement || document.mozFullScreenElement || document.msFullscreenElement);
    if (initialFullscreen) {
      setIsFullscreen(true);
    }

    if (!isElevated && hasStarted && isFullscreen) {
      interval = setInterval(() => {
        if (!document.fullscreenElement && !document.webkitFullscreenElement && !document.mozFullScreenElement && !document.msFullscreenElement) {
          dispatchTelemetry("fullscreen_dropped", { action: "force_check_failed" });
        }
      }, 5000);
    }

    const handleVisibilityChange = () => {
      if (document.hidden) {
        setTabViolations(prev => {
          const newCount = prev + 1;
          dispatchTelemetry("visibility_spoof_suspected", {
            action: "focus_lost",
            warning_count: newCount
          });
          return newCount;
        });
      }
    };

    const handleBeforeUnload = (e) => {
      e.preventDefault();
      e.returnValue = "You are actively in a contest. Leaving will discard unsaved code.";
      return e.returnValue;
    };

    if (!isElevated && hasStarted) {
      document.addEventListener('visibilitychange', handleVisibilityChange);
      window.addEventListener('beforeunload', handleBeforeUnload);
    }

    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      document.removeEventListener('webkitfullscreenchange', handleFullscreenChange);
      document.removeEventListener('mozfullscreenchange', handleFullscreenChange);
      document.removeEventListener('MSFullscreenChange', handleFullscreenChange);
      clearInterval(interval);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('beforeunload', handleBeforeUnload);
    };
  }, [isElevated, hasStarted, isFullscreen, dispatchTelemetry]);

  return { isElevated, isFullscreen, tabViolations, toggleFullscreen };
}
