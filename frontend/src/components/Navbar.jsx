import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode'; //

export default function Navbar() {
  const [isLoggedIn, setIsLoggedIn] = useState(!!localStorage.getItem('token'));
  const [userRole, setUserRole] = useState(localStorage.getItem('role'));
  const [isOnboarded, setIsOnboarded] = useState(false); //

  // Function to derive onboarding status from the current token
  const checkOnboardingStatus = () => {
    const token = localStorage.getItem('token');
    if (token) {
      try {
        const decoded = jwtDecode(token);
        setIsOnboarded(!!decoded.is_onboarded); //
      } catch (e) {
        setIsOnboarded(false); //
      }
    } else {
      setIsOnboarded(false);
    }
  };

  useEffect(() => {
    // Initial check on mount
    checkOnboardingStatus();

    const syncAuthState = () => {
      setIsLoggedIn(!!localStorage.getItem('token'));
      setUserRole(localStorage.getItem('role'));
      checkOnboardingStatus(); // Re-check status when storage changes
    };

    window.addEventListener('storage', syncAuthState);
    return () => window.removeEventListener('storage', syncAuthState);
  }, []);

  const handleLogout = () => {
    localStorage.clear(); 
    setIsLoggedIn(false);
    setUserRole(null);
    setIsOnboarded(false);
    window.location.href = '/login'; 
  };

  const hasElevatedAccess = userRole?.toLowerCase() === 'admin' || userRole?.toLowerCase() === 'professor';

  return (
    <nav className="p-4 bg-dark-surface border-b border-dark-border flex justify-between items-center shadow-md">
      <div className="flex items-center space-x-8">
        <Link to="/" className="text-xl font-bold text-blue-500 tracking-wide hover:text-blue-400 transition">
          CampusCompile
        </Link>
        
        {/* 👇 CRITICAL CHANGE: 
            Navigation links are ONLY visible if the user is logged in AND onboarded.
            This prevents students from clicking "Problems" to escape the onboarding screen. 
        */}
        {isLoggedIn && isOnboarded && (
          <div className="flex space-x-6 text-sm font-semibold text-gray-300">
            <Link to="/problems" className="hover:text-white transition">Problems</Link>
            
            {hasElevatedAccess && (
              <Link to="/add-problem" className="text-green-400 hover:text-green-300 transition flex items-center gap-1">
                <span>+</span> Forge Problem
              </Link>
            )}
          </div>
        )}
      </div>

      <div className="space-x-4 flex items-center">
        {isLoggedIn ? (
          <button 
            onClick={handleLogout} 
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