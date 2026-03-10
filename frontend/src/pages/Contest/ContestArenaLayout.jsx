import React from 'react';
import { Outlet, Link, useParams, useLocation } from 'react-router-dom';
import { LogOut } from 'lucide-react';

export default function ContestArenaLayout() {
  const { id: contestId } = useParams();
  const location = useLocation();
  
  // 👇 Check if the user is a Professor or Admin
  const userRole = localStorage.getItem('role');
  const isElevated = userRole === 'admin' || userRole === 'professor';

  const isLeaderboard = location.pathname.includes('leaderboard');

  return (
    <div className="min-h-screen bg-dark-bg text-white relative">
      
      {/* 👇 THE ESCAPE HATCH (Only visible to Admins & Professors) */}
      {isElevated && (
        <Link 
          to="/contests" 
          className="absolute top-4 right-4 z-50 flex items-center gap-2 bg-red-900/50 hover:bg-red-600 text-red-200 hover:text-white px-4 py-2 rounded border border-red-700 transition-colors shadow-lg text-sm font-bold tracking-wider"
        >
          <LogOut size={16} />
          Exit to Workspace
        </Link>
      )}

      {/* Hub Navigation Tabs */}
      <div className="bg-[#1e1e1e] border-b border-dark-border px-8 pt-6 flex gap-6">
        <Link 
          to={`/contests/${contestId}/arena`} 
          className={`pb-3 font-bold transition-colors ${!isLeaderboard ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-white'}`}
        >
          Arena Problems
        </Link>
        <Link 
          to={`/contests/${contestId}/arena/leaderboard`} 
          className={`pb-3 font-bold transition-colors ${isLeaderboard ? 'text-yellow-400 border-b-2 border-yellow-400' : 'text-gray-400 hover:text-white'}`}
        >
          Live Leaderboard
        </Link>
      </div>

      {/* Renders either ContestProblems or Leaderboard based on the URL */}
      <div className="p-8">
        <Outlet />
      </div>
    </div>
  );
}