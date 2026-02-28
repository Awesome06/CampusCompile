import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';

import Navbar from './components/Navbar';

// Page Imports
import Arena from './pages/Arena';
import Login from './pages/Login';
import Landing from './pages/Landing';         
import ProblemList from './pages/ProblemList'; 
import AddProblem from './pages/AddProblem';
import OAuthSuccess from './pages/OAuthSuccess';
import Onboarding from './pages/Onboarding';

// The Bouncer
const ProtectedRoute = ({ children }) => {
  const token = localStorage.getItem('token');
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  return children;
};

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-dark-bg text-dark-text font-sans">
        
        {/* 👇 Drop the Navbar right here! */}
        <Navbar />

        {/* Route Configuration */}
        <Routes>
          {/* Public Routes */}
          <Route path="/" element={<Landing />} />
          <Route path="/problems" element={<ProblemList />} />
          <Route path="/login" element={<Login />} />
          <Route path="/oauth-success" element={<OAuthSuccess />} />
          
          {/* Protected Routes */}
          <Route path="/onboarding" element={<ProtectedRoute><Onboarding /></ProtectedRoute>} />
          <Route path="/arena/:id" element={<ProtectedRoute><Arena /></ProtectedRoute>} />
          <Route path="/add-problem" element={<ProtectedRoute><AddProblem /></ProtectedRoute>} />
        </Routes>
        
      </div>
    </BrowserRouter>
  );
}

export default App;