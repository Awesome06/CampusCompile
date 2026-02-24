import React, { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
import axios from 'axios';
import Arena from './Arena';

// --- PLACEHOLDER COMPONENTS ---
// In the future, we will move these into their own separate files (e.g., Landing.jsx, Problems.jsx)

function Landing() {
  return (
    <div className="flex flex-col items-center justify-center h-[calc(100vh-61px)]">
      <h1 className="text-5xl font-bold text-white mb-6">Welcome to CampusCompile</h1>
      <p className="text-xl text-gray-400 mb-8">The ultimate competitive programming arena.</p>
      <Link to="/problems" className="bg-dark-accent text-white px-6 py-3 rounded-lg font-bold hover:bg-blue-600 transition">
        Enter the Arena
      </Link>
    </div>
  );
}

function ProblemList() {
  // State to hold the data from our Go API
  const [problems, setProblems] = useState([]);
  const [loading, setLoading] = useState(true);

  // Fetch data when the component loads
  useEffect(() => {
    axios.get('http://localhost:8080/api/problems')
      .then((response) => {
        setProblems(response.data); // Save the JSON array to state
        setLoading(false);
      })
      .catch((error) => {
        console.error("Error fetching problems:", error);
        setLoading(false);
      });
  }, []);

  return (
    <div className="p-8 max-w-6xl mx-auto">
      <h2 className="text-3xl font-bold text-white mb-6">Problem Repository</h2>
      <div className="bg-dark-surface border border-dark-border rounded-lg p-4">
        
        {/* Table Header */}
        <div className="flex justify-between items-center py-3 border-b border-dark-border text-gray-400 font-semibold">
          <span className="w-1/2">Title</span>
          <span className="w-1/4">Difficulty</span>
          <span className="w-1/4 text-right">Action</span>
        </div>

        {/* Loading State */}
        {loading && <div className="text-center py-8 text-gray-400">Loading problems from server...</div>}

        {/* Dynamic Data Mapping */}
        {!loading && problems.map((prob) => (
          <div key={prob.problem_id} className="flex justify-between items-center py-4 text-white border-b border-dark-border last:border-0 hover:bg-[#363636] px-2 rounded transition">
            <span className="w-1/2 font-medium">{prob.title}</span>
            <span className={`w-1/4 font-semibold ${
              prob.difficulty === 'Easy' ? 'text-green-400' : 
              prob.difficulty === 'Medium' ? 'text-yellow-400' : 'text-red-400'
            }`}>
              {prob.difficulty}
            </span>
            <span className="w-1/4 text-right">
              {/* This link dynamically inserts the real database UUID into the URL! */}
              <Link to={`/arena/${prob.problem_id}`} className="bg-dark-accent px-4 py-1.5 rounded hover:bg-blue-600 transition text-sm font-bold shadow">
                Solve
              </Link>
            </span>
          </div>
        ))}

      </div>
    </div>
  );
}

function Leaderboard() {
  return (
    <div className="p-8 max-w-4xl mx-auto">
      <h2 className="text-3xl font-bold text-white mb-6">Campus Leaderboard</h2>
      <div className="bg-dark-surface border border-dark-border rounded-lg p-8 text-center text-gray-400">
        Global rankings will be displayed here.
      </div>
    </div>
  );
}

// --- MAIN APP COMPONENT ---

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-dark-bg text-dark-text font-sans">
        
        {/* Global Navigation Bar */}
        <nav className="p-4 bg-dark-surface border-b border-dark-border flex justify-between items-center">
          <div className="flex items-center space-x-8">
            <Link to="/" className="text-xl font-bold text-dark-accent tracking-wide hover:text-blue-400 transition">
              CampusCompile
            </Link>
            
            {/* Page Links */}
            <div className="flex space-x-6 text-sm font-semibold text-gray-300">
              <Link to="/problems" className="hover:text-white transition">Problems</Link>
              <Link to="/leaderboard" className="hover:text-white transition">Leaderboard</Link>
            </div>
          </div>

          <div className="space-x-4">
            <span className="text-sm font-medium text-gray-400">Welcome, Knight</span>
          </div>
        </nav>

        {/* Route Configuration */}
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/problems" element={<ProblemList />} />
          <Route path="/leaderboard" element={<Leaderboard />} />
          
          {/* Notice the :id parameter! This allows us to load specific problems dynamically */}
          <Route path="/arena/:id" element={<Arena />} />
        </Routes>
        
      </div>
    </BrowserRouter>
  );
}

export default App;