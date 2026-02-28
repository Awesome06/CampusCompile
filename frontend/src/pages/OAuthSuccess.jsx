import React, { useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';

export default function OAuthSuccess() {
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    // 1. Parse the URL parameters sent by Go's HandleAzureCallback
    const params = new URLSearchParams(location.search);
    const token = params.get('token');
    const role = params.get('role');
    const onboarded = params.get('onboarded');

    if (token) {
      // 2. Save the credentials securely in the browser
      localStorage.setItem('token', token);
      
      // Store the role (admin, professor, student) for UI rendering checks
      if (role) {
        localStorage.setItem('role', role);
      }

      // 3. The Traffic Cop Logic
      // We use window.location.href here instead of navigate() to force a hard 
      // page reload. This ensures App.jsx completely re-mounts and updates the 
      // Navigation bar to show the correct logged-in state and "+ Forge Problem" button.
      if (onboarded === 'true') {
        // Veteran user -> Send them straight to the Arena
        window.location.href = '/problems';
      } else {
        // Brand new user -> Force them to fill out their Batch/Section
        window.location.href = '/onboarding';
      }
    } else {
      // If someone manually types /oauth-success without a token, kick them out
      navigate('/login');
    }
  }, [navigate, location]);

  return (
    <div className="flex items-center justify-center h-[calc(100vh-61px)] bg-dark-bg">
      <div className="text-white text-xl font-bold animate-pulse flex flex-col items-center gap-4">
        {/* Optional: Add a spinner or a shield icon here */}
        <svg className="w-12 h-12 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        Authenticating with Microsoft...
      </div>
    </div>
  );
}