import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import axios from 'axios';

export default function Register() {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const navigate = useNavigate();

const handleRegister = async (e) => {
    e.preventDefault();
    setError('');
    
    try {
      // 1. Create the account in PostgreSQL
      await axios.post('http://localhost:8080/api/auth/register', {
        username,
        email,
        password
      });

      // 2. Automatically log them in using the exact same credentials!
      const loginResponse = await axios.post('http://localhost:8080/api/auth/login', {
        email,
        password
      });

      // 3. Save the JWT token to Local Storage
      localStorage.setItem('token', loginResponse.data.token);
      
      // 4. Hard redirect to the Problem Repository (this also fixes the Navbar update!)
      window.location.href = '/problems';

    } catch (err) {
      if (!email.includes('@campus.edu.in')) {
        setError('Only valid university emails (@campus.edu.in) are allowed.');
      } else if (err.response && err.response.data && err.response.data.error) {
        // 👇 This grabs the exact error message from your Go API!
        setError(`Server Error: ${err.response.data.error}`);
      } else {
        setError('Registration failed. Username or Email might already exist.');
      } 
    }
  };

  return (
    <div className="flex items-center justify-center h-[calc(100vh-61px)] bg-dark-bg">
      <div className="bg-dark-surface p-8 rounded-lg border border-dark-border w-96 shadow-2xl">
        <h2 className="text-2xl font-bold text-white mb-6 text-center tracking-wide">Forge Your Profile</h2>
        
        {error && <div className="bg-red-900/50 border border-red-500 text-red-400 p-2 rounded mb-4 text-sm text-center">{error}</div>}

        <form onSubmit={handleRegister} className="flex flex-col gap-4">
          <div>
            <label className="block text-gray-400 text-sm font-bold mb-2">Username</label>
            <input 
              type="text" 
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full bg-dark-bg text-white px-3 py-2 rounded border border-dark-border focus:outline-none focus:border-dark-accent transition" 
              required 
            />
          </div>
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
          <button type="submit" className="w-full bg-dark-success text-white font-bold py-2 px-4 rounded hover:bg-green-600 transition mt-2">
            Register
          </button>
        </form>

        <p className="text-gray-400 text-sm text-center mt-6">
          Already a Knight? <Link to="/login" className="text-dark-accent hover:text-blue-400">Login</Link>
        </p>
      </div>
    </div>
  );
}