import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import api from '../services/api'; 
import Button from '../components/ui/Button';

export default function ProblemList() {
  const [problems, setProblems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState('public'); // 'public' or 'faculty'
  const navigate = useNavigate();

  const userRole = localStorage.getItem('role');
  const canAddProblem = userRole?.toLowerCase() === 'admin' || userRole?.toLowerCase() === 'professor';

  useEffect(() => {
    setLoading(true);
    // Dynamically switch the API endpoint based on the selected tab
    const endpoint = viewMode === 'faculty' ? '/faculty/problems' : '/problems';
    
    api.get(endpoint)
      .then((response) => {
        setProblems(response.data);
        setLoading(false);
      })
      .catch((error) => {
        console.error("Error fetching problems:", error);
        setLoading(false);
      });
  }, [viewMode]);

  return (
    <div className="p-8 max-w-6xl mx-auto">
      
      {/* Header & Tabs */}
      <div className="relative flex justify-center items-center mb-8 h-10">
        
        {/* If the user is a professor/admin, show tabs to toggle views */}
        {canAddProblem ? (
          <div className="flex bg-[#1e1e1e] rounded-lg p-1 border border-dark-border shadow-lg">
            <button 
              onClick={() => setViewMode('public')}
              className={`px-6 py-2 text-sm font-bold rounded-md transition-all ${viewMode === 'public' ? 'bg-dark-accent text-white shadow' : 'text-gray-500 hover:text-gray-300'}`}
            >
              Public Arena
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
            <span className="w-12 text-left font-bold text-gray-500">{index + 1}</span>
            
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
            <span className={`w-1/4 font-semibold ${
                prob.difficulty === 'Easy' ? 'text-green-400' : 
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
      </div>
    </div>
  );
}