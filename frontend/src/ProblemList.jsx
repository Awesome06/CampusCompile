import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import axios from 'axios';

export default function ProblemList() {
  const [problems, setProblems] = useState([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  // 👇 1. Check user role from localStorage
  const userRole = localStorage.getItem('role');
  const canAddProblem = userRole === 'admin' || userRole === 'professor';

  useEffect(() => {
    axios.get('http://localhost:8080/api/problems')
      .then((response) => {
        setProblems(response.data);
        setLoading(false);
      })
      .catch((error) => {
        console.error("Error fetching problems:", error);
        setLoading(false);
      });
  }, []);

  return (
    <div className="p-8 max-w-6xl mx-auto">
      
      {/* 👇 2. Header Row with Title and Conditional Button */}
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-3xl font-bold text-white">Problem Repository</h2>
        
        {/* The button only renders if canAddProblem is true */}
        {canAddProblem && (
          <button 
            onClick={() => navigate('/add-problem')}
            className="bg-green-600 hover:bg-green-500 text-white font-bold py-2 px-4 rounded shadow-lg transition"
          >
            + Add Problem
          </button>
        )}
      </div>

      <div className="bg-[#1e1e1e] border border-dark-border rounded-lg p-4 shadow-lg">
        
        <div className="flex justify-between items-center py-3 border-b border-dark-border text-gray-400 font-semibold px-2">
          <span className="w-12 text-left">#</span>
          <span className="w-1/2">Title</span>
          <span className="w-1/4">Difficulty</span>
          <span className="w-1/4 text-right">Action</span>
        </div>

        {loading && <div className="text-center py-8 text-gray-400">Loading problems...</div>}
        
        {!loading && problems.map((prob, index) => (
          <div key={prob.problem_id} className="flex justify-between items-center py-4 text-white border-b border-dark-border last:border-0 hover:bg-[#2a2a2a] px-2 rounded transition">
            <span className="w-12 text-left font-bold text-gray-500">{index + 1}</span>
            <span className="w-1/2 font-medium">{prob.title}</span>
            <span className={`w-1/4 font-semibold ${
                prob.difficulty === 'Easy' ? 'text-green-400' : 
                prob.difficulty === 'Medium' ? 'text-yellow-400' : 
                'text-red-400'
              }`}>
              {prob.difficulty}
            </span>
            <span className="w-1/4 text-right">
              <Link to={`/arena/${prob.problem_id}`} className="bg-blue-600 px-4 py-1.5 rounded hover:bg-blue-500 transition text-sm font-bold shadow">
                Solve
              </Link>
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}