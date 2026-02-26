import React, { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
import axios from 'axios';
import Arena from './Arena';
import Login from './Login';       // <-- New Import
import Register from './Register'; // <-- New Import
import { Navigate } from 'react-router-dom';

const ProtectedRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  
  if (!token) {
    // Redirect to login if no token is found
    return <Navigate to="/login" replace />;
  }

  return children;
};

// --- PLACEHOLDER COMPONENTS ---
function Landing() {
  return (
    <div className="flex flex-col items-center justify-center h-[calc(100vh-61px)]">
      
      {/* 👇 NEW: Centered Logo above the text 👇 */}
      <img 
        src="/Logo_small.png" 
        alt="CampusCompile Logo" 
        className="w-48 h-auto mb-8 drop-shadow-2xl" 
      />
      
      <h1 className="text-5xl font-bold text-white mb-6">Welcome to CampusCompile</h1>
      <p className="text-xl text-gray-400 mb-8">The ultimate competitive programming arena.</p>
      
      <Link to="/problems" className="bg-dark-accent text-white px-6 py-3 rounded-lg font-bold hover:bg-blue-600 transition shadow-lg">
        Enter the Arena
      </Link>
      
    </div>
  );
}

function ProblemList() {
  const [problems, setProblems] = useState([]);
  const [loading, setLoading] = useState(true);

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
      <h2 className="text-3xl font-bold text-white mb-6">Problem Repository</h2>
      <div className="bg-dark-surface border border-dark-border rounded-lg p-4 shadow-lg">
        
        {/* Updated Header with a "#" column */}
        <div className="flex justify-between items-center py-3 border-b border-dark-border text-gray-400 font-semibold px-2">
          <span className="w-12 text-left">#</span>
          <span className="w-1/2">Title</span>
          <span className="w-1/4">Difficulty</span>
          <span className="w-1/4 text-right">Action</span>
        </div>

        {loading && <div className="text-center py-8 text-gray-400">Loading problems...</div>}
        
        {/* Updated Mapping logic to use (prob, index) */}
        {!loading && problems.map((prob, index) => (
          <div key={prob.problem_id} className="flex justify-between items-center py-4 text-white border-b border-dark-border last:border-0 hover:bg-[#363636] px-2 rounded transition">
            
            {/* The numeric index starting at 1 */}
            <span className="w-12 text-left font-bold text-gray-500">{index + 1}</span>
            
            <span className="w-1/2 font-medium">{prob.title}</span>
            <span className={`w-1/4 font-semibold ${prob.difficulty === 'Easy' ? 'text-green-400' : 'text-red-400'}`}>
              {prob.difficulty}
            </span>
            <span className="w-1/4 text-right">
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

// --- MAIN APP COMPONENT ---
function App() {
  // Check if a token exists in the browser's storage
  const isLoggedIn = !!localStorage.getItem('token');

  const handleLogout = () => {
    localStorage.removeItem('token');
    window.location.href = '/login'; // Force a full page reload to clear state
  };

  return (
    <BrowserRouter>
      <div className="min-h-screen bg-dark-bg text-dark-text font-sans">
        
        {/* Global Navigation Bar */}
        <nav className="p-4 bg-dark-surface border-b border-dark-border flex justify-between items-center">
          <div className="flex items-center space-x-8">
            <Link to="/" className="text-xl font-bold text-dark-accent tracking-wide hover:text-blue-400 transition">
              CampusCompile
            </Link>
            <div className="flex space-x-6 text-sm font-semibold text-gray-300">
              <Link to="/problems" className="hover:text-white transition">Problems</Link>
            </div>
          </div>

          <div className="space-x-4 flex items-center">
            {isLoggedIn ? (
              <button onClick={handleLogout} className="text-sm font-medium text-red-400 hover:text-red-300 transition">
                Logout
              </button>
            ) : (
              <>
                <Link to="/login" className="text-sm font-medium text-gray-300 hover:text-white transition">Login</Link>
                <Link to="/register" className="text-sm font-medium bg-dark-accent px-4 py-1.5 rounded hover:bg-blue-600 transition text-white">Register</Link>
              </>
            )}
          </div>
        </nav>

        {/* Route Configuration */}
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/problems" element={<ProblemList />} />
          <Route 
            path="/arena/:id" 
            element={
              <ProtectedRoute>
                <Arena />
              </ProtectedRoute>
            } 
          />
          <Route path="/login" element={<Login />} />       {/* <-- New Route */}
          <Route path="/register" element={<Register />} /> {/* <-- New Route */}
        </Routes>
        
      </div>
    </BrowserRouter>
  );
}

export default App;