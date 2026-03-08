import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import api from '../services/api'; 
import Button from '../components/ui/Button';
import { useAuth } from '../context/AuthContext';

export default function ContestList() {
  const [contests, setContests] = useState([]);
  const [loading, setLoading] = useState(true);
  const { currentUser } = useAuth();
  const navigate = useNavigate();

  const isElevated = currentUser?.role === 'admin' || currentUser?.role === 'professor';
  
  // Default tabs based on role
  const [viewMode, setViewMode] = useState(isElevated ? 'public' : 'upcoming'); 

  useEffect(() => {
    setLoading(true);
    api.get('/contests')
      .then((response) => {
        setContests(response.data || []);
        setLoading(false);
      })
      .catch((error) => {
        console.error("Error fetching contests:", error);
        setLoading(false);
      });
  }, []);

  // Time & Status Evaluation
  const now = new Date();
  
  // Filter logic based on the active tab
  const filteredContests = contests.filter(contest => {
    const startTime = new Date(contest.start_time);
    const endTime = new Date(contest.end_time);

    if (isElevated) {
      if (viewMode === 'public') return contest.is_public === true;
      if (viewMode === 'faculty') return contest.author_id === currentUser?.id || !contest.is_public || currentUser?.role === 'admin';
    } else {
      if (viewMode === 'upcoming') return endTime > now;
      if (viewMode === 'past') return endTime <= now;
    }
    return true;
  });

  // Helper to determine the visual status badge
  const getStatusBadge = (contest) => {
    if (!contest.is_public) {
      return <span className="bg-yellow-900/20 text-yellow-500 border border-yellow-700/50 text-[10px] px-2 py-0.5 rounded font-bold tracking-wider">○ DRAFT</span>;
    }
    
    const startTime = new Date(contest.start_time);
    const endTime = new Date(contest.end_time);

    if (now < startTime) {
      return <span className="bg-blue-900/20 text-blue-400 border border-blue-700/50 text-[10px] px-2 py-0.5 rounded font-bold tracking-wider">UPCOMING</span>;
    } else if (now >= startTime && now <= endTime) {
      return <span className="bg-green-900/20 text-green-400 border border-green-700/50 text-[10px] px-2 py-0.5 rounded font-bold tracking-wider animate-pulse">● LIVE NOW</span>;
    } else {
      return <span className="bg-gray-800 text-gray-400 border border-gray-600 text-[10px] px-2 py-0.5 rounded font-bold tracking-wider">ENDED</span>;
    }
  };

  // Helper to render Demographic Access Tags
  const renderAccessTags = (rules) => {
    if (!rules) return <span className="text-xs text-gray-500 font-mono bg-[#2a2a2a] px-2 py-1 rounded">🌍 Global Arena</span>;
    
    const tags = [];
    if (rules.allowed_courses?.length > 0) tags.push(...rules.allowed_courses);
    if (rules.allowed_departments?.length > 0) tags.push(...rules.allowed_departments);
    if (rules.allowed_graduation_years?.length > 0) tags.push(...rules.allowed_graduation_years.map(y => `'${y.toString().slice(-2)}`));
    
    if (tags.length === 0) return <span className="text-xs text-gray-500 font-mono bg-[#2a2a2a] px-2 py-1 rounded">🌍 Global Arena</span>;

    return (
      <div className="flex gap-1 flex-wrap">
        {tags.slice(0, 3).map((tag, i) => (
          <span key={i} className="text-[10px] text-gray-400 font-mono bg-[#2a2a2a] px-1.5 py-0.5 rounded border border-dark-border">{tag}</span>
        ))}
        {tags.length > 3 && <span className="text-[10px] text-gray-500 font-mono px-1">+{tags.length - 3}</span>}
      </div>
    );
  };

  return (
    <div className="p-8 max-w-6xl mx-auto">
      
      {/* Header & Dynamic Tabs */}
      <div className="relative flex justify-center items-center mb-8 h-10">
        <div className="flex bg-[#1e1e1e] rounded-lg p-1 border border-dark-border shadow-lg">
          
          {/* Faculty Tabs */}
          {isElevated && (
            <>
              <button onClick={() => setViewMode('public')} className={`px-6 py-2 text-sm font-bold rounded-md transition-all ${viewMode === 'public' ? 'bg-dark-accent text-white shadow' : 'text-gray-500 hover:text-gray-300'}`}>
                Contests Published
              </button>
              <button onClick={() => setViewMode('faculty')} className={`px-6 py-2 text-sm font-bold rounded-md transition-all ${viewMode === 'faculty' ? 'bg-dark-accent text-white shadow' : 'text-gray-500 hover:text-gray-300'}`}>
                Contests Created
              </button>
            </>
          )}

          {/* Student Tabs */}
          {!isElevated && (
            <>
              <button onClick={() => setViewMode('upcoming')} className={`px-6 py-2 text-sm font-bold rounded-md transition-all ${viewMode === 'upcoming' ? 'bg-dark-accent text-white shadow' : 'text-gray-500 hover:text-gray-300'}`}>
                Active & Upcoming
              </button>
              <button onClick={() => setViewMode('past')} className={`px-6 py-2 text-sm font-bold rounded-md transition-all ${viewMode === 'past' ? 'bg-dark-accent text-white shadow' : 'text-gray-500 hover:text-gray-300'}`}>
                Past Contests
              </button>
            </>
          )}
        </div>
        
        {/* Right-Pinned Create Button */}
        {isElevated && (
          <div className="absolute right-0">
            <Button onClick={() => navigate('/add-contest')} variant="success" size="sm" className="border border-green-600 hover:border-green-500">
              <span>+</span> Create Contest
            </Button>
          </div>
        )}
      </div>

      {/* Contest List Render */}
      <div className="bg-[#1e1e1e] border border-dark-border rounded-lg p-4 shadow-lg">
        <div className="flex justify-between items-center py-3 border-b border-dark-border text-gray-400 font-semibold px-4">
          <span className="w-1/2">Contest Details</span>
          <span className="w-1/4">Clearance Needed</span>
          <span className="w-1/4 text-right">Action</span>
        </div>

        {loading && <div className="text-center py-8 text-gray-400 animate-pulse font-mono">Scanning arena servers...</div>}
        
        {!loading && filteredContests.length === 0 && (
          <div className="text-center py-10 text-gray-500 italic border-b border-dark-border last:border-0">
            No contests found in this category.
          </div>
        )}
        
        {!loading && filteredContests.map((contest) => (
          <div key={contest.contest_id} className="flex justify-between items-center py-5 text-white border-b border-dark-border last:border-0 hover:bg-[#252525] px-4 rounded transition group">
            
            {/* Title, Organization & Status */}
            <div className="w-1/2 flex flex-col items-start gap-2">
              <div className="flex items-center gap-3">
                <span className="font-bold text-lg">{contest.title}</span>
                {getStatusBadge(contest)}
              </div>
              <div className="text-xs text-gray-400 font-mono flex gap-3">
                <span>🏢 {contest.host_organization || "CampusCompile Official"}</span>
                <span>📅 {new Date(contest.start_time).toLocaleDateString()}</span>
              </div>
            </div>

            {/* Demographic Tags */}
            <div className="w-1/4">
              {renderAccessTags(contest.access_rules)}
            </div>

            {/* Actions */}
            <span className="w-1/4 text-right flex justify-end gap-2">
               {isElevated && viewMode === 'faculty' && (
                <Link to={`/edit-contest/${contest.contest_id}`} className="bg-[#2a2a2a] px-4 py-2 rounded border border-dark-border hover:bg-gray-700 transition text-sm font-bold text-gray-300 shadow-sm">
                  Configure
                </Link>
              )}
              <Link to={`/contests/${contest.contest_id}`} className="bg-blue-600 px-6 py-2 rounded hover:bg-blue-500 transition text-sm font-bold shadow-sm text-white flex items-center gap-2">
                {now < new Date(contest.start_time) ? 'View Details' : 'Enter Arena'}
              </Link>
            </span>

          </div>
        ))}
      </div>
    </div>
  );
}