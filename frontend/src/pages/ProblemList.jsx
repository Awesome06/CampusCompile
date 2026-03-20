import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import api from '../services/api';
import Button from '../components/ui/Button';

export default function ProblemList() {
  const [problems, setProblems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState('public'); // 'public' or 'faculty'
  const [searchQuery, setSearchQuery] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 25;
  const [hasMore, setHasMore] = useState(true);
  const navigate = useNavigate();

  const userRole = localStorage.getItem('role');
  const canAddProblem = userRole?.toLowerCase() === 'admin' || userRole?.toLowerCase() === 'professor';

  const fetchProblems = (currentOffset, query) => {
    if (currentOffset === 0) setLoading(true);
    const endpoint = viewMode === 'faculty' ? '/faculty/problems' : '/problems';

    api.get(endpoint, { params: { limit, offset: currentOffset, search: query } })
      .then((response) => {
        const data = response.data || [];
        setProblems(data);
        setHasMore(data.length === limit);
        setLoading(false);
      })
      .catch((error) => {
        console.error("Error fetching problems:", error);
        setLoading(false);
      });
  };

  useEffect(() => {
    const delayDebounceFn = setTimeout(() => {
      setOffset(0);
      setHasMore(true);
      fetchProblems(0, searchQuery);
    }, 400);

    return () => clearTimeout(delayDebounceFn);
  }, [searchQuery, viewMode]);

  const handlePrev = () => {
    if (offset >= limit) {
      const nextOffset = offset - limit;
      setOffset(nextOffset);
      fetchProblems(nextOffset, searchQuery);
    }
  };

  const handleNext = () => {
    if (hasMore) {
      const nextOffset = offset + limit;
      setOffset(nextOffset);
      fetchProblems(nextOffset, searchQuery);
    }
  };

  useEffect(() => {
    const handleKeyDown = (e) => {
      // Don't trigger if user is typing in the search bar
      if (document.activeElement.tagName === 'INPUT' || document.activeElement.tagName === 'TEXTAREA') return;
      if (e.key === 'ArrowLeft') {
        handlePrev();
      } else if (e.key === 'ArrowRight') {
        handleNext();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [offset, hasMore, searchQuery, viewMode]);

  const displayStart = problems.length > 0 ? offset + 1 : 0;
  const displayEnd = offset + problems.length;

  return (
    <div className="p-8 max-w-6xl mx-auto">

      {/* Header & Tabs */}
      <div className="relative flex justify-center items-center mb-8 h-10 w-full">

        <div className="absolute left-0 w-1/3 min-w-[200px]">
          <input
            type="text"
            placeholder="Search problems..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full bg-[#1e1e1e] border border-dark-border rounded-md px-4 py-2 text-white focus:outline-none focus:border-blue-500 transition shadow-sm text-sm"
          />
        </div>

        {/* If the user is a professor/admin, show tabs to toggle views */}
        {canAddProblem ? (
          <div className="flex bg-[#1e1e1e] rounded-lg p-1 border border-dark-border shadow-lg">
            <button
              onClick={() => setViewMode('public')}
              className={`px-6 py-2 text-sm font-bold rounded-md transition-all ${viewMode === 'public' ? 'bg-dark-accent text-white shadow' : 'text-gray-500 hover:text-gray-300'}`}
            >
              Published Problems
            </button>
            <button
              onClick={() => setViewMode('faculty')}
              className={`px-6 py-2 text-sm font-bold rounded-md transition-all ${viewMode === 'faculty' ? 'bg-dark-accent text-white shadow' : 'text-gray-500 hover:text-gray-300'}`}
            >
              Problems Created
            </button>
          </div>
        ) : (
          <h2 className="text-3xl font-bold text-white whitespace-nowrap">Problem Repository</h2>
        )}

        {/* Right-Pinned Button */}
        {canAddProblem && (
          <div className="absolute right-0">
            <Button
              onClick={() => navigate('/add-problem')}
              variant="success"
              size="sm"
              className="border border-green-600 hover:border-green-500"
            >
              <span>+</span> Forge Problem
            </Button>
          </div>
        )}
      </div>

      <div className="bg-[#1e1e1e] border border-dark-border rounded-lg p-4 shadow-lg">

        <div className="flex justify-between items-center py-3 border-b border-dark-border text-gray-400 font-semibold px-2">
          <span className="w-12 text-left">#</span>
          <span className="w-1/2">Title</span>
          <span className="w-1/4">Difficulty</span>
          <span className="w-1/4 text-right">Action</span>
        </div>

        {loading && <div className="text-center py-8 text-gray-400 animate-pulse">Loading arena data...</div>}

        {!loading && problems.length === 0 && (
          <div className="text-center py-10 text-gray-500 italic border-b border-dark-border last:border-0">
            {viewMode === 'faculty' ? "You haven't forged any problems yet." : "No problems have been forged yet."}
          </div>
        )}

        {!loading && problems.map((prob, index) => (
          <div key={prob.problem_id} className="flex justify-between items-center py-4 text-white border-b border-dark-border last:border-0 hover:bg-[#2a2a2a] px-2 rounded transition relative">
            <span className="w-12 text-left font-bold text-gray-500">{offset + index + 1}</span>

            {/* Title & Status Badges */}
            <div className="w-1/2 font-medium flex items-center gap-3">
              <span className="truncate">{prob.title}</span>

              {/* Faculty Workspace Status Indicators */}
              {viewMode === 'faculty' && (
                prob.is_public ? (
                  <span className="bg-green-900/20 text-green-400 border border-green-700/50 text-[10px] px-2 py-0.5 rounded font-bold tracking-wider whitespace-nowrap">
                    ● LIVE
                  </span>
                ) : (
                  <span className="bg-yellow-900/20 text-yellow-500 border border-yellow-700/50 text-[10px] px-2 py-0.5 rounded font-bold tracking-wider whitespace-nowrap">
                    ○ DRAFT
                  </span>
                )
              )}
            </div>

            {/* Difficulty */}
            <span className={`w-1/4 font-semibold ${prob.difficulty === 'Easy' ? 'text-green-400' :
                prob.difficulty === 'Medium' ? 'text-yellow-400' :
                  'text-red-400'
              }`}>
              {prob.difficulty}
            </span>

            {/* Actions */}
            <span className="w-1/4 text-right flex justify-end gap-2">
              {viewMode === 'faculty' && (
                <Link to={`/edit-problem/${prob.problem_id}`} className="bg-[#2a2a2a] px-4 py-1.5 rounded border border-dark-border hover:bg-gray-700 transition text-sm font-bold text-gray-300 shadow-sm">
                  Edit
                </Link>
              )}
              <Link to={`/arena/${prob.problem_id}`} className="bg-blue-600 px-4 py-1.5 rounded hover:bg-blue-500 transition text-sm font-bold shadow-sm text-white">
                Solve
              </Link>
            </span>
          </div>
        ))}

        {!loading && (hasMore || offset > 0) && (
          <div className="flex justify-between items-center mt-6 pt-4 border-t border-dark-border">
            <Button onClick={handlePrev} variant="outline" disabled={offset === 0} className="w-1/5 text-gray-300 border-dark-border hover:bg-[#2a2a2a] disabled:opacity-30 disabled:cursor-not-allowed">
              &larr; Prev
            </Button>
            <span className="w-3/5 text-center text-gray-400 text-sm font-mono">
              Showing {displayStart} - {displayEnd}
            </span>
            <Button onClick={handleNext} variant="outline" disabled={!hasMore} className="w-1/5 text-gray-300 border-dark-border hover:bg-[#2a2a2a] disabled:opacity-30 disabled:cursor-not-allowed">
              Next &rarr;
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}