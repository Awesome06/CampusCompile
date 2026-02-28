import React from 'react';
import { Link } from 'react-router-dom';

export default function Landing() {
  // Check local storage to see who is visiting (set by the Go backend via OAuthSuccess)
  const token = localStorage.getItem('token');
  const role = localStorage.getItem('role');

  // Normalize the role check to perfectly match the Go backend's ENUMs
  const hasElevatedAccess = role?.toLowerCase() === 'admin' || role?.toLowerCase() === 'professor';

  return (
    <div className="flex flex-col items-center justify-center h-[calc(100vh-61px)] px-4">
      
      <img 
        src="/Logo_small.png" 
        alt="CampusCompile Logo" 
        className="w-48 h-auto mb-8 drop-shadow-2xl" 
      />
      
      <h1 className="text-5xl font-bold text-white mb-6 text-center">
        Welcome to CampusCompile
      </h1>
      <p className="text-xl text-gray-400 mb-10 text-center max-w-2xl">
        The ultimate competitive programming arena for students and professors.
      </p>
      
      {/* Dynamic Button Rendering */}
      <div className="flex flex-wrap justify-center gap-4">
        
        {!token ? (
          <Link to="/login" className="bg-dark-surface border border-dark-border text-white px-8 py-3 rounded-lg font-bold hover:bg-[#2a2a2a] transition shadow-lg">
            Login
          </Link>
        ) : (
          <>
            <Link to="/problems" className="bg-dark-accent text-white px-8 py-3 rounded-lg font-bold hover:bg-blue-600 transition shadow-lg">
              Enter the Arena
            </Link>
            
            {/* This perfectly respects the role calculated by your Go Regex Engine 
              during the initial Microsoft Graph API handshake. 
            */}
            {hasElevatedAccess && (
              <Link to="/add-problem" className="bg-green-700 text-white px-8 py-3 rounded-lg font-bold hover:bg-green-600 transition shadow-lg border border-green-600 hover:border-green-500 flex items-center gap-2">
                <span>+</span> Forge Problem
              </Link>
            )}
          </>
        )}
        
      </div>
    </div>
  );
}