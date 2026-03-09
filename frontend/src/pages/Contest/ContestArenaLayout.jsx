import React from 'react';
import { Outlet, Link, useLocation, useParams } from 'react-router-dom';
import { List, Trophy } from 'lucide-react';

export default function ContestArenaLayout() {
  const { id } = useParams();
  const location = useLocation();

  // Determine which tab is active based on the URL
  const isLeaderboard = location.pathname.includes('leaderboard');

  return (
    <div className="min-h-screen bg-dark-bg text-gray-200 font-sans">
      {/* Contest Hub Navbar */}
      <div className="bg-[#1e1e1e] border-b border-dark-border px-6 py-4 flex items-center justify-between sticky top-0 z-50 shadow-md">
        <h1 className="text-xl font-bold text-white flex items-center gap-2">
          <Trophy className="text-yellow-500" size={24} />
          Contest Arena
        </h1>
        
        {/* Tab Navigation */}
        <div className="flex bg-[#2a2a2a] rounded-lg p-1 border border-dark-border">
          <Link
            to={`/contests/${id}/arena`}
            className={`flex items-center gap-2 px-6 py-2 rounded-md transition text-sm font-bold tracking-wide ${
              !isLeaderboard ? 'bg-dark-accent text-white shadow' : 'text-gray-400 hover:text-white'
            }`}
          >
            <List size={16} /> Problems
          </Link>
          <Link
            to={`/contests/${id}/arena/leaderboard`}
            className={`flex items-center gap-2 px-6 py-2 rounded-md transition text-sm font-bold tracking-wide ${
              isLeaderboard ? 'bg-dark-accent text-white shadow' : 'text-gray-400 hover:text-white'
            }`}
          >
            <Trophy size={16} /> Leaderboard
          </Link>
        </div>
      </div>

      {/* Render either ContestProblems or Leaderboard here */}
      <div className="p-6 max-w-7xl mx-auto">
        <Outlet />
      </div>
    </div>
  );
}