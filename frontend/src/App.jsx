import React, { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Link, Navigate } from 'react-router-dom';
import Arena from './Arena';
import Login from './Login';
import Landing from './Landing';         
import ProblemList from './ProblemList'; 
import AddProblem from './AddProblem';
import OAuthSuccess from './OAuthSuccess';
import Onboarding from './Onboarding';

const ProtectedRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  return children;
};

function App() {
  // Initialize state directly from localStorage
  const [isLoggedIn, setIsLoggedIn] = useState(!!localStorage.getItem('token'));
  const [userRole, setUserRole] = useState(localStorage.getItem('role'));

  // Sync state if localStorage changes (handles logout/login re-renders)
  useEffect(() => {
    const syncAuthState = () => {
      setIsLoggedIn(!!localStorage.getItem('token'));
      setUserRole(localStorage.getItem('role'));
    };

    window.addEventListener('storage', syncAuthState);
    return () => window.removeEventListener('storage', syncAuthState);
  }, []);

  const handleLogout = () => {
    localStorage.clear(); 
    setIsLoggedIn(false);
    setUserRole(null);
    window.location.href = '/login'; 
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
                <Link to="/login" className="text-sm font-medium bg-[#0078D4] px-4 py-1.5 rounded hover:bg-[#005ea6] transition text-white flex items-center gap-2">
                  Sign In
                </Link>
              </>
            )}
          </div>
        </nav>

        {/* Route Configuration */}
        <Routes>
          {/* Public Routes */}
          <Route path="/" element={<Landing />} />
          <Route path="/problems" element={<ProblemList />} />
          <Route path="/login" element={<Login />} />
          
          {/* SSO Callback (Must be public to catch Microsoft's redirect) */}
          <Route path="/oauth-success" element={<OAuthSuccess />} />
          
          {/* Protected Routes (Requires JWT Token) */}
          <Route 
            path="/onboarding" 
            element={
              <ProtectedRoute>
                <Onboarding />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="/arena/:id" 
            element={
              <ProtectedRoute>
                <Arena />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="/add-problem" 
            element={
              <ProtectedRoute>
                <AddProblem />
              </ProtectedRoute>
            } 
          />
        </Routes>
        
      </div>
    </BrowserRouter>
  );
}

export default App;