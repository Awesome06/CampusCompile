import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import { AuthProvider, useAuth } from './context/AuthContext'; 

import Navbar from './components/Navbar';

// Page Imports
import Arena from './pages/Arena/Arena';
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

// Phase 4 Contest Redesign Imports
import ContestArenaLayout from './pages/Contest/ContestArenaLayout';
import ContestProblems from './pages/Contest/ContestProblems';

// The Bouncer
const ProtectedRoute = ({ children, requireOnboarding = true }) => {
  const { token, isLoading } = useAuth(); 

  // Prevent redirect flickering while context initializes
  if (isLoading) {
    return <div className="flex h-screen items-center justify-center bg-dark-bg text-white font-mono">Loading Session...</div>;
  }

  if (!token) return <Navigate to="/login" replace />;

  try {
    const decoded = jwtDecode(token);
    
    // If you ARE onboarded, but trying to access the /onboarding page...
    if (!requireOnboarding && decoded.is_onboarded) {
      // ...immediately send you to the problems list instead
      return <Navigate to="/problems" replace />;
    }

    // If the page REQUIRES onboarding and you haven't done it...
    if (requireOnboarding && !decoded.is_onboarded) {
      // ...lock you into the onboarding page
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

          {/* Route Configuration */}
          <Routes>
            <Route path="/" element={<Landing />} />
            <Route path="/login" element={<Login />} />
            <Route path="/oauth-success" element={<OAuthSuccess />} />
            
            {/* Onboarding doesn't require "onboarding" to be true, but requires a token */}
            <Route path="/onboarding" element={
              <ProtectedRoute requireOnboarding={false}>
                <Onboarding />
              </ProtectedRoute>
            } />

            {/* STANDARD PRACTICE ARENA (No Anti-Cheat, No Draft Wipes) */}
            <Route path="/arena/:id" element={<ProtectedRoute><Arena isContest={false} /></ProtectedRoute>} />

            <Route path="/problems" element={<ProtectedRoute><ProblemList /></ProtectedRoute>} />
            <Route path="/add-problem" element={<ProtectedRoute><AddProblem /></ProtectedRoute>} />
            <Route path="/edit-problem/:id" element={<ProtectedRoute><EditProblem /></ProtectedRoute>} />

            <Route path="/contests" element={<ProtectedRoute><ContestList /></ProtectedRoute>} />
            <Route path="/add-contest" element={<ProtectedRoute><AddContest /></ProtectedRoute>} />
            <Route path="/edit-contest/:id" element={<ProtectedRoute><EditContest /></ProtectedRoute>} />
            
            {/* =========================================
                CONTEST HUB & RESTRICTED ARENA
                ========================================= */}
                
            {/* The Hub (Problems List & Live Leaderboard) */}
            <Route path="/contests/:id/arena" element={<ProtectedRoute><ContestArenaLayout /></ProtectedRoute>}>
              <Route index element={<ContestProblems />} />
              <Route path="leaderboard" element={<Leaderboard />} />
            </Route>

            {/* The Restricted Editor (Anti-Cheat Traps ARMED) */}
            <Route 
              path="/contests/:id/problem/:problemId" 
              element={<ProtectedRoute><Arena isContest={true} /></ProtectedRoute>} 
            />

          </Routes>
          
        </div>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;