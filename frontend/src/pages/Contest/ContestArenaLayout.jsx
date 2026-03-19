import React, { useState, useEffect, useRef } from 'react';
import { Outlet, Link, useParams, useLocation, useNavigate } from 'react-router-dom';
import { LogOut, Maximize, AlertTriangle, CheckCircle } from 'lucide-react';
import useAntiCheat from '../../hooks/useAntiCheat'; 

export default function ContestArenaLayout() {
  const { id: contestId } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  
  const getIsFullscreen = () => !!(
    document.fullscreenElement ||
    document.mozFullScreenElement ||
    document.webkitFullscreenElement ||
    document.msFullscreenElement
  );

  const [isFullscreen, setIsFullscreen] = useState(getIsFullscreen);
  
  const userRole = localStorage.getItem('role');
  const isElevated = userRole === 'admin' || userRole === 'professor';

  // Tracks if the student has actively started the contest
  const [hasEnteredArena, setHasEnteredArena] = useState(!isElevated && getIsFullscreen()); 
  const [showFinishModal, setShowFinishModal] = useState(false);
  const [tabViolations, setTabViolations] = useState(0); 
  
  const isLeaderboard = location.pathname.includes('leaderboard');

  // Anti-cheat only boots up for students AFTER they enter the arena
  const { dispatchTelemetry } = useAntiCheat(contestId, !isElevated && hasEnteredArena);

  // Track time spent outside fullscreen
  const leaveTimeRef = useRef(null);
  const [timeAway, setTimeAway] = useState(0);

  useEffect(() => {
    const handleFullscreenChange = () => {
      const isCurrentlyFullscreen = !!(
        document.fullscreenElement ||
        document.mozFullScreenElement ||
        document.webkitFullscreenElement ||
        document.msFullscreenElement
      );

      setIsFullscreen(isCurrentlyFullscreen);
      
      if (!isElevated) {
        if (isCurrentlyFullscreen && !hasEnteredArena) {
          // First time entering the arena
          setHasEnteredArena(true);
        } else if (isCurrentlyFullscreen && hasEnteredArena && leaveTimeRef.current) {
          // Returning to the arena after dropping fullscreen
          const durationSecs = Math.floor((Date.now() - leaveTimeRef.current) / 1000);
          dispatchTelemetry("fullscreen_dropped", { duration_seconds: durationSecs });
          leaveTimeRef.current = null;
          setTimeAway(0);
        } else if (!isCurrentlyFullscreen && hasEnteredArena) {
          // Dropping out of the arena
          leaveTimeRef.current = Date.now();
          setTimeAway(0);
        }
      }
    };
    
    // Listen for standard and vendor-prefixed fullscreen events
    document.addEventListener('fullscreenchange', handleFullscreenChange);
    document.addEventListener('webkitfullscreenchange', handleFullscreenChange);
    document.addEventListener('mozfullscreenchange', handleFullscreenChange);
    document.addEventListener('MSFullscreenChange', handleFullscreenChange);

    // Live timer for the UI Lockout screen
    let interval;
    if (!isElevated && hasEnteredArena && !isFullscreen) {
      interval = setInterval(() => {
        if (leaveTimeRef.current) {
          setTimeAway(Math.floor((Date.now() - leaveTimeRef.current) / 1000));
        }
      }, 1000);
    }

    // PROCTORING LOGIC (Only applies to students AFTER they start)
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

    if (!isElevated && hasEnteredArena) {
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
  }, [isElevated, hasEnteredArena, isFullscreen, dispatchTelemetry]);

  const toggleFullscreen = async () => {
    try {
      const element = document.documentElement;
      
      const isCurrentlyFullscreen = !!(
        document.fullscreenElement ||
        document.mozFullScreenElement ||
        document.webkitFullscreenElement ||
        document.msFullscreenElement
      );

      if (!isCurrentlyFullscreen) {
        if (element.requestFullscreen) {
          await element.requestFullscreen();
        } else if (element.msRequestFullscreen) {
          await element.msRequestFullscreen();
        } else if (element.mozRequestFullScreen) {
          await element.mozRequestFullScreen();
        } else if (element.webkitRequestFullscreen) {
          await element.webkitRequestFullscreen();
        }
      } else {
        if (document.exitFullscreen) {
          await document.exitFullscreen();
        } else if (document.msExitFullscreen) {
          await document.msExitFullscreen();
        } else if (document.mozCancelFullScreen) {
          await document.mozCancelFullScreen();
        } else if (document.webkitExitFullscreen) {
          await document.webkitExitFullscreen();
        }
      }
    } catch (err) {
      console.error("Error attempting to toggle fullscreen:", err);
    }
  };

  // The graceful exit process
  const handleFinishContest = async () => {
    if (document.fullscreenElement && document.exitFullscreen) {
      await document.exitFullscreen();
    }
    navigate('/contests'); // Return to workspace
  };

  // Normal Layout Render
  return (
    <div 
      className={`bg-dark-bg text-white relative flex flex-col ${isFullscreen || (!isElevated && hasEnteredArena && !isFullscreen) ? 'h-screen w-screen overflow-hidden' : 'min-h-screen'}`}
    >
      {/* THE LOCKOUT SCREEN */}
      {!isElevated && hasEnteredArena && !isFullscreen && (
        <div className="fixed inset-0 h-screen w-screen bg-red-950 text-white flex flex-col items-center justify-center p-8 z-[9999]">
          <AlertTriangle size={80} className="text-yellow-500 mb-6 animate-pulse" />
          <h1 className="text-4xl font-bold mb-4 text-center">Environment Lockout</h1>
          <p className="text-xl mb-4 text-center text-red-200 max-w-2xl">
            You have exited the secure Arena. To maintain academic integrity, you must remain in Fullscreen mode. Continuing to leave the environment will flag your submission.
          </p>
          
          <div className="bg-red-900 border border-red-500 rounded-lg p-4 mb-8 text-center shadow-xl">
            <span className="block text-red-300 font-mono text-sm tracking-widest uppercase mb-1">Time Outside Arena</span>
            <span className="text-5xl font-mono font-bold text-white tracking-widest">{timeAway}s</span>
            <span className="block text-red-400 text-xs mt-2 italic">This incident is being recorded</span>
          </div>

          <button 
            onClick={toggleFullscreen} 
            className="bg-white text-red-900 px-8 py-4 rounded font-bold text-xl hover:bg-gray-200 transition shadow-lg"
          >
            Acknowledge & Return to Arena
          </button>
        </div>
      )}
      {/* Finish Contest Confirmation Modal */}
      {showFinishModal && (
        <div className="fixed inset-0 z-[100] bg-black/80 flex items-center justify-center backdrop-blur-sm">
          <div className="bg-dark-surface border border-dark-border p-8 rounded-lg max-w-md w-full shadow-2xl">
            <h2 className="text-2xl font-bold mb-4 text-white flex items-center gap-2">
              <CheckCircle className="text-green-500" /> Finish Contest?
            </h2>
            <p className="text-gray-300 mb-8">
              Are you sure you want to finish the contest and exit the arena? Ensure all your code is submitted before leaving.
            </p>
            <div className="flex justify-end gap-4">
              <button 
                onClick={() => setShowFinishModal(false)}
                className="px-4 py-2 rounded font-bold text-gray-300 hover:bg-gray-700 transition"
              >
                Cancel
              </button>
              <button 
                onClick={handleFinishContest}
                className="px-4 py-2 rounded font-bold bg-green-600 hover:bg-green-500 text-white transition"
              >
                Confirm & Exit
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Warning Banner for Tab Switchers */}
      {!isElevated && tabViolations > 0 && (
        <div className="bg-red-600 text-white text-center py-1 text-sm font-bold flex justify-center items-center gap-2 flex-shrink-0">
          <AlertTriangle size={14} />
          Warning: Focus lost {tabViolations} time(s). This activity is being recorded.
        </div>
      )}

      {/* Top Navbar Area */}
      <div className="bg-[#1e1e1e] border-b border-dark-border px-8 pt-4 flex justify-between items-end flex-shrink-0 relative z-40">
        
        <div className="flex gap-6">
          <Link 
            to={`/contests/${contestId}/arena`} 
            className={`pb-3 font-bold transition-colors ${!isLeaderboard ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-white'}`}
          >
            Arena Problems
          </Link>
          <Link 
            to={`/contests/${contestId}/arena/leaderboard`} 
            className={`pb-3 font-bold transition-colors ${isLeaderboard ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-white'}`}
          >
            Live Leaderboard
          </Link>
        </div>

        {/* Controls */}
        <div className="flex items-center gap-3 pb-3">
          
          {/* Finish Button (Students Only, After Starting) */}
          {!isElevated && hasEnteredArena && (
            <button 
              onClick={() => setShowFinishModal(true)}
              className="flex items-center gap-2 bg-green-700 hover:bg-green-600 text-white px-4 py-2 rounded border border-green-500 transition-colors shadow-lg text-sm font-bold tracking-wider"
            >
              <CheckCircle size={16} />
              Finish Contest
            </button>
          )}

          {/* Escape Hatch (Admins/Professors Only) */}
          {isElevated && (
            <>
              <Link 
                to="/contests" 
                className="flex items-center gap-2 bg-red-900/50 hover:bg-red-600 text-red-200 hover:text-white px-4 py-2 rounded border border-red-700 transition-colors shadow-lg text-sm font-bold tracking-wider"
              >
                <LogOut size={16} />
                Exit to Workspace
              </Link>
            </>
          )}
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-auto relative z-0">
        
        {/* The Pre-Contest Lobby Overlay */}
        {!isElevated && !hasEnteredArena && (
          <div className="absolute inset-0 z-50 bg-black/70 backdrop-blur-sm flex flex-col items-center justify-center">
            <div className="bg-dark-surface p-8 rounded-xl border border-blue-900 shadow-2xl text-center max-w-lg">
              <h2 className="text-3xl font-bold text-white mb-4">Ready to Begin?</h2>
              <p className="text-gray-300 mb-8">
                You must enter secure Arena Mode to view problems and submit code. Once started, exiting fullscreen will trigger a security lockout.
              </p>
              <button 
                onClick={toggleFullscreen}
                className="bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded font-bold text-xl transition-colors w-full flex items-center justify-center gap-3 shadow-lg hover:scale-105 transform duration-200"
              >
                <Maximize size={24} />
                Enter Secure Arena
              </button>
            </div>
          </div>
        )}
        
        <Outlet />
      </div>
    </div>
  );
}