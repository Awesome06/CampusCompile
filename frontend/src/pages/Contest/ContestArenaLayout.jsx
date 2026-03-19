import React, { useState, useEffect, useRef } from 'react';
import { Outlet, Link, useParams, useLocation, useNavigate } from 'react-router-dom';
import { LogOut, Maximize, AlertTriangle, CheckCircle } from 'lucide-react';
import useSecureArena from '../../hooks/useSecureArena';

export default function ContestArenaLayout() {
  const { id: contestId } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  
  const [hasEnteredArena, setHasEnteredArena] = useState(false); 
  const [showFinishModal, setShowFinishModal] = useState(false);
  
  const isLeaderboard = location.pathname.includes('leaderboard');

  const { isElevated, isFullscreen, tabViolations, toggleFullscreen } = useSecureArena(contestId, hasEnteredArena);

  const leaveTimeRef = useRef(null);
  const [timeAway, setTimeAway] = useState(0);

  useEffect(() => {
    // If student, and not elevated, and is fullscreen, then arena entered
    if (!isElevated && isFullscreen && !hasEnteredArena) {
      setHasEnteredArena(true);
    }
  }, [isElevated, isFullscreen, hasEnteredArena]);

  useEffect(() => {
    let interval;
    if (!isElevated && hasEnteredArena) {
        if (!isFullscreen) {
            if (!leaveTimeRef.current) leaveTimeRef.current = Date.now();
            interval = setInterval(() => {
                setTimeAway(Math.floor((Date.now() - leaveTimeRef.current) / 1000));
            }, 1000);
        } else {
            leaveTimeRef.current = null;
            setTimeAway(0);
        }
    }
    return () => clearInterval(interval);
  }, [isElevated, hasEnteredArena, isFullscreen]);

  const handleFinishContest = async () => {
    if (document.fullscreenElement && document.exitFullscreen) {
      await document.exitFullscreen();
    }
    navigate('/contests');
  };

  return (
    <div className={`bg-dark-bg text-white relative flex flex-col ${isFullscreen || (!isElevated && hasEnteredArena && !isFullscreen) ? 'h-screen w-screen overflow-hidden' : 'min-h-screen'}`}>
      
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
          <button onClick={toggleFullscreen} className="bg-white text-red-900 px-8 py-4 rounded font-bold text-xl hover:bg-gray-200 transition shadow-lg">
            Acknowledge & Return to Arena
          </button>
        </div>
      )}

      {showFinishModal && (
        <div className="fixed inset-0 z-[100] bg-black/80 flex items-center justify-center backdrop-blur-sm">
          <div className="bg-dark-surface border border-dark-border p-8 rounded-lg max-w-md w-full shadow-2xl">
            <h2 className="text-2xl font-bold mb-4 text-white flex items-center gap-2"><CheckCircle className="text-green-500" /> Finish Contest?</h2>
            <p className="text-gray-300 mb-8">Are you sure you want to finish the contest and exit the arena? Ensure all your code is submitted before leaving.</p>
            <div className="flex justify-end gap-4">
              <button onClick={() => setShowFinishModal(false)} className="px-4 py-2 rounded font-bold text-gray-300 hover:bg-gray-700 transition">Cancel</button>
              <button onClick={handleFinishContest} className="px-4 py-2 rounded font-bold bg-green-600 hover:bg-green-500 text-white transition">Confirm & Exit</button>
            </div>
          </div>
        </div>
      )}

      {!isElevated && tabViolations > 0 && (
        <div className="bg-red-600 text-white text-center py-1 text-sm font-bold flex justify-center items-center gap-2 flex-shrink-0">
          <AlertTriangle size={14} /> Warning: Focus lost {tabViolations} time(s). This activity is being recorded.
        </div>
      )}

      <div className="bg-[#1e1e1e] border-b border-dark-border px-8 pt-4 flex justify-between items-end flex-shrink-0 relative z-40">
        <div className="flex gap-6">
          <Link to={`/contests/${contestId}/arena`} className={`pb-3 font-bold transition-colors ${!isLeaderboard ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-white'}`}>Arena Problems</Link>
          <Link to={`/contests/${contestId}/arena/leaderboard`} className={`pb-3 font-bold transition-colors ${isLeaderboard ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-white'}`}>Live Leaderboard</Link>
        </div>
        <div className="flex items-center gap-3 pb-3">
          {!isElevated && hasEnteredArena && (
            <button onClick={() => setShowFinishModal(true)} className="flex items-center gap-2 bg-green-700 hover:bg-green-600 text-white px-4 py-2 rounded border border-green-500 transition-colors shadow-lg text-sm font-bold tracking-wider">
              <CheckCircle size={16} /> Finish Contest
            </button>
          )}
          {isElevated && (
            <Link to="/contests" className="flex items-center gap-2 bg-red-900/50 hover:bg-red-600 text-red-200 hover:text-white px-4 py-2 rounded border border-red-700 transition-colors shadow-lg text-sm font-bold tracking-wider">
              <LogOut size={16} /> Exit to Workspace
            </Link>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-auto relative z-0">
        {!isElevated && !hasEnteredArena && (
          <div className="absolute inset-0 z-50 bg-black/70 backdrop-blur-sm flex flex-col items-center justify-center">
            <div className="bg-dark-surface p-8 rounded-xl border border-blue-900 shadow-2xl text-center max-w-lg">
              <h2 className="text-3xl font-bold text-white mb-4">Ready to Begin?</h2>
              <p className="text-gray-300 mb-8">You must enter secure Arena Mode to view problems and submit code. Once started, exiting fullscreen will trigger a security lockout.</p>
              <button onClick={toggleFullscreen} className="bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded font-bold text-xl transition-colors w-full flex items-center justify-center gap-3 shadow-lg hover:scale-105 transform duration-200">
                <Maximize size={24} /> Enter Secure Arena
              </button>
            </div>
          </div>
        )}
        <Outlet />
      </div>
    </div>
  );
}