import React, { useEffect } from 'react';

export default function Login() {
  // Wipe stale keys when landing on the login page to ensure a clean slate
  useEffect(() => {
    localStorage.removeItem('token');
    localStorage.removeItem('role');
  }, []);

  const handleSSOLogin = () => {
    // 🛑 CRITICAL: Do not use Axios here! 
    // OAuth requires a hard browser redirect to leave the app and hit Microsoft's servers.
    // This perfectly triggers the handlers.HandleAzureLogin function in your Go backend.
    window.location.href = 'http://localhost:8080/api/auth/login';
  };

  return (
    <div className="flex items-center justify-center h-[calc(100vh-61px)] bg-dark-bg">
      <div className="bg-dark-surface p-8 rounded-lg border border-dark-border w-96 shadow-2xl text-center">
        <img src="/Logo_small.png" alt="Logo" className="w-24 mx-auto mb-6" />
        <h2 className="text-2xl font-bold text-white mb-2 tracking-wide">Enter the Arena</h2>
        <p className="text-gray-400 text-sm mb-8">Access restricted to Bennett University students and faculty.</p>

        <button 
          onClick={handleSSOLogin} 
          className="w-full bg-[#0078D4] text-white font-bold py-3 px-4 rounded hover:bg-[#005ea6] transition flex items-center justify-center gap-3 shadow-lg"
        >
          {/* Official Microsoft Logo SVG */}
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 48 48">
            <path fill="#f3f3f3" d="M24 24h24v24H24z"/>
            <path fill="#f35325" d="M0 0h22v22H0z"/>
            <path fill="#81bc06" d="M24 0h24v22H24z"/>
            <path fill="#05a6f0" d="M0 24h22v24H0z"/>
            <path fill="#ffba08" d="M24 24h24v24H24z"/>
          </svg>
          Sign in with Microsoft
        </button>
      </div>
    </div>
  );
}