import React, { useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';

export default function OAuthSuccess() {
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    // 1. Parse the URL parameters Go sent us
    const params = new URLSearchParams(location.search);
    const token = params.get('token');
    const role = params.get('role');
    const onboarded = params.get('onboarded');

    if (token) {
      // 2. Save the credentials
      localStorage.setItem('token', token);
      localStorage.setItem('role', role);

      // 3. The Traffic Cop Logic
      if (onboarded === 'true') {
        // Veteran user -> Send them to the Arena
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
      <div className="text-white text-xl font-bold animate-pulse">
        Authenticating with Microsoft...
      </div>
    </div>
  );
}