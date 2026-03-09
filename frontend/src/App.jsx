import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import { AuthProvider, useAuth } from './context/AuthContext'; 

import Navbar from './components/Navbar';

// Page Imports
import Login from './pages/Login';
import Landing from './pages/Landing';         
import ProblemList from './pages/ProblemList'; 
import AddProblem from './pages/AddProblem';
import EditProblem from './pages/EditProblem'; 
import OAuthSuccess from './pages/OAuthSuccess';
import Onboarding from './pages/Onboarding';
import ContestList from './pages/ContestList';
import EditContest from './pages/EditContest';
import AddContest from './pages/AddContest';
import Leaderboard from './pages/Leaderboard';

// Phase 4 Imports
import ContestArenaLayout from './pages/Contest/ContestArenaLayout';
import ContestProblems from './pages/Contest/ContestProblems';
import PracticeArena from './pages/Arena/PracticeArena';
import ContestArena from './pages/Arena/ContestArena';

// The Bouncer
const ProtectedRoute = ({ children, requireOnboarding = true }) => {
  const { token, isLoading } = useAuth(); 

  if (isLoading) {
    return <div className="flex h-screen items-center justify-center bg-dark-bg text-white font-mono">Loading Session...</div>;
  }

  if (!token) return <Navigate to="/login" replace />;

  try {
    const decoded = jwtDecode(token);
    
    if (!requireOnboarding && decoded.is_onboarded) {
      return <Navigate to="/problems" replace />;
    }

    if (requireOnboarding && !decoded.is_onboarded) {
      return <Navigate to="/onboarding" replace />;
    }
  } catch (error) {
    return <Navigate to="/login" replace />;
  }

  return children;
};

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <div className="min-h-screen bg-dark-bg text-dark-text font-sans">
          
          <Navbar />

          <Routes>
            {/* Public Routes */}
            <Route path="/" element={<Landing />} />
            <Route path="/login" element={<Login />} />
            <Route path="/oauth-success" element={<OAuthSuccess />} />
            
            <Route path="/onboarding" element={
              <ProtectedRoute requireOnboarding={false}>
                <Onboarding />
              </ProtectedRoute>
            } />

            {/* DEDICATED PRACTICE ROUTE */}
            <Route path="/arena/:id" element={<ProtectedRoute><PracticeArena /></ProtectedRoute>} />

            {/* Repositories */}
            <Route path="/problems" element={<ProtectedRoute><ProblemList /></ProtectedRoute>} />
            <Route path="/add-problem" element={<ProtectedRoute><AddProblem /></ProtectedRoute>} />
            <Route path="/edit-problem/:id" element={<ProtectedRoute><EditProblem /></ProtectedRoute>} />

            <Route path="/contests" element={<ProtectedRoute><ContestList /></ProtectedRoute>} />
            <Route path="/add-contest" element={<ProtectedRoute><AddContest /></ProtectedRoute>} />
            <Route path="/edit-contest/:id" element={<ProtectedRoute><EditContest /></ProtectedRoute>} />
            
            {/* Contest Hub Layout */}
            <Route path="/contests/:id/arena" element={<ProtectedRoute><ContestArenaLayout /></ProtectedRoute>}>
              <Route index element={<ContestProblems />} />
              <Route path="leaderboard" element={<Leaderboard />} />
            </Route>

            {/* DEDICATED CONTEST EDITOR ROUTE */}
            <Route 
              path="/contests/:id/problem/:problemId" 
              element={<ProtectedRoute><ContestArena /></ProtectedRoute>} 
            />

          </Routes>
          
        </div>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;