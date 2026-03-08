import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import { AuthProvider, useAuth } from './context/AuthContext'; // 👈 Imported Context

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

// The Bouncer
const ProtectedRoute = ({ children, requireOnboarding = true }) => {
  const { token, isLoading } = useAuth(); // 👈 Now pulling from Context

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
    // 👇 Wrap the entire app in the AuthProvider
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

            {/* These strictly require the user to be onboarded */}
            <Route path="/arena/:id" element={<ProtectedRoute><Arena /></ProtectedRoute>} />

            <Route path="/problems" element={<ProtectedRoute><ProblemList /></ProtectedRoute>} />
            <Route path="/add-problem" element={<ProtectedRoute><AddProblem /></ProtectedRoute>} />
            <Route path="/edit-problem/:id" element={<ProtectedRoute><EditProblem /></ProtectedRoute>} />

            <Route path="/contests" element={<ProtectedRoute><ContestList /></ProtectedRoute>} />
            <Route path="/add-contest" element={<ProtectedRoute><AddContest /></ProtectedRoute>} />
            <Route path="/edit-contest/:id" element={<ProtectedRoute><EditContest /></ProtectedRoute>} />
            
          </Routes>
          
        </div>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;