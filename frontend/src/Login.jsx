import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import axios from 'axios';

export default function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleLogin = async (e) => {
    e.preventDefault();
    setError('');
    
    try {
      const response = await axios.post('http://localhost:8080/api/auth/login', {
        email,
        password
      });

      // 🏆 SUCCESS! Save the token to the browser's Local Storage
      localStorage.setItem('token', response.data.token);
      
      // Redirect the knight to the problem repository
      window.location.href = '/problems';
    } catch (err) {
      setError('Invalid email or password. Are you sure you are a Knight?');
    }
  };

  return (
    <div className="flex items-center justify-center h-[calc(100vh-61px)] bg-dark-bg">
      <div className="bg-dark-surface p-8 rounded-lg border border-dark-border w-96 shadow-2xl">
        <h2 className="text-2xl font-bold text-white mb-6 text-center tracking-wide">Enter the Arena</h2>
        
        {error && <div className="bg-red-900/50 border border-red-500 text-red-400 p-2 rounded mb-4 text-sm text-center">{error}</div>}

        <form onSubmit={handleLogin} className="flex flex-col gap-4">
          <div>
            <label className="block text-gray-400 text-sm font-bold mb-2">Email</label>
            <input 
              type="email" 
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full bg-dark-bg text-white px-3 py-2 rounded border border-dark-border focus:outline-none focus:border-dark-accent transition" 
              required 
            />
          </div>
          <div>
            <label className="block text-gray-400 text-sm font-bold mb-2">Password</label>
            <input 
              type="password" 
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-dark-bg text-white px-3 py-2 rounded border border-dark-border focus:outline-none focus:border-dark-accent transition" 
              required 
            />
          </div>
          <button type="submit" className="w-full bg-dark-accent text-white font-bold py-2 px-4 rounded hover:bg-blue-600 transition mt-2">
            Login
          </button>
        </form>

        <p className="text-gray-400 text-sm text-center mt-6">
          Don't have an account? <Link to="/register" className="text-dark-accent hover:text-blue-400">Register</Link>
        </p>
      </div>
    </div>
  );
}