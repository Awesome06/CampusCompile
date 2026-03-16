import React, { useEffect } from 'react';
import Button from '../components/ui/Button';
import MicrosoftIcon from '../components/icons/MicrosoftIcon';
import api from '../services/api';

export default function Login() {
  useEffect(() => {
    localStorage.removeItem('token');
    localStorage.removeItem('role');
  }, []);

  const handleSSOLogin = () => {
    const baseUrl = api.defaults.baseURL;
    window.location.href = `${baseUrl}/auth/login`;
  };

  return (
    <div className="flex items-center justify-center h-[calc(100vh-61px)] bg-dark-bg">
      <div className="bg-dark-surface p-8 rounded-lg border border-dark-border w-96 shadow-2xl text-center">
        <img src="/Logo_small.png" alt="Logo" className="w-24 mx-auto mb-6" />
        <h2 className="text-2xl font-bold text-white mb-2 tracking-wide">Enter the Arena</h2>
        <p className="text-gray-400 text-sm mb-8">Access restricted to Bennett University students and faculty.</p>

        <Button onClick={handleSSOLogin} variant="microsoft">
          <MicrosoftIcon />
          Sign in with Microsoft
        </Button>
      </div>
    </div>
  );
}