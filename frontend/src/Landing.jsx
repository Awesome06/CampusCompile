import React from 'react';
import { Link } from 'react-router-dom';

export default function Landing() {
  // Check local storage to see who is visiting
  const token = localStorage.getItem('token');
  const role = localStorage.getItem('role');

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
          <>
            <Link to="/login" className="bg-dark-surface border border-dark-border text-white px-8 py-3 rounded-lg font-bold hover:bg-[#2a2a2a] transition shadow-lg">
              Login
            </Link>
          </>
        ) : (
          <>
            <Link to="/problems" className="bg-dark-accent text-white px-8 py-3 rounded-lg font-bold hover:bg-blue-600 transition shadow-lg">
              Enter the Arena
            </Link>
            
            {(role === 'professor' || role === 'admin') && (
              <Link to="/add-problem" className="bg-purple-600 text-white px-8 py-3 rounded-lg font-bold hover:bg-purple-500 transition shadow-lg border border-purple-500 hover:border-purple-400">
                + Add Problem
              </Link>
            )}
          </>
        )}
        
      </div>
    </div>
  );
}