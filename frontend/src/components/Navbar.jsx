import React from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function Navbar() {
  const { isLoggedIn, currentUser, logout } = useAuth(); 
  const isOnboarded = currentUser?.isOnboarded;

  return (
    <nav className="p-4 bg-dark-surface border-b border-dark-border flex justify-between items-center shadow-md">
      <div className="flex items-center space-x-8">
        <Link to="/" className="text-xl font-bold text-blue-500 tracking-wide hover:text-blue-400 transition">
          CampusCompile
        </Link>
        
        {isLoggedIn && isOnboarded && (
          <div className="flex space-x-6 text-sm font-semibold text-gray-300 items-center">
            <Link to="/problems" className="hover:text-white transition">Problems</Link>
            <Link to="/contests" className="hover:text-white transition">Contests</Link>
          </div>
        )}
      </div>

      <div className="space-x-4 flex items-center">
        {isLoggedIn ? (
          <button 
            onClick={logout} 
            className="text-sm font-medium text-red-400 hover:text-red-300 transition"
          >
            Logout
          </button>
        ) : (
          <Link 
            to="/login" 
            className="text-sm font-medium bg-[#0078D4] px-4 py-1.5 rounded hover:bg-[#005ea6] transition text-white flex items-center gap-2 shadow-lg"
          >
            Sign In
          </Link>
        )}
      </div>
    </nav>
  );
}