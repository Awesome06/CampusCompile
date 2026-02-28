import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../services/api';

export default function Onboarding() {
  const navigate = useNavigate();
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const [formData, setFormData] = useState({
    username: '',
    course: 'B.Tech',
    department: 'CSE',
    course_year: 1,
    batch: '',
    section: '',
    student_group: ''
  });

  useEffect(() => {
    if (!localStorage.getItem('token')) {
      navigate('/login');
    }
  }, [navigate]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      await api.post('/auth/onboard', {
        ...formData,
        course_year: parseInt(formData.course_year, 10) 
      });

      window.location.href = '/problems';
      
    } catch (err) {
      console.error("Onboarding error:", err);
      if (err.response?.data?.error) {
        setError(err.response.data.error); 
      } else {
        setError('Failed to save profile. Please try again.');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-[calc(100vh-61px)] flex items-center justify-center bg-dark-bg py-10 px-4">
      <div className="bg-[#1e1e1e] p-8 rounded-lg border border-dark-border w-full max-w-2xl shadow-2xl">
        <h2 className="text-3xl font-bold text-white mb-2 text-center tracking-wide">Complete Your Profile</h2>
        <p className="text-gray-400 text-sm mb-8 text-center">
          Forge your identity. This data determines your contest eligibility and leaderboard groups.
        </p>
        
        {error && (
          <div className="bg-red-900/50 border border-red-500 text-red-400 p-3 rounded mb-6 text-sm text-center font-bold">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Username Row */}
          <div>
            <label className="block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wide">Arena Username</label>
            <input 
              type="text" 
              name="username"
              value={formData.username}
              onChange={handleChange}
              className="w-full bg-[#2a2a2a] text-white px-4 py-3 rounded border border-dark-border focus:outline-none focus:border-blue-500 transition" 
              placeholder="e.g., CodeNinja99"
              required 
              maxLength="50"
            />
            <p className="text-xs text-gray-500 mt-1">This will be your public handle on the leaderboards.</p>
          </div>

          <div className="border-t border-dark-border my-6"></div>

          {/* Academic Metadata Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            
            <div>
              <label className="block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wide">Course</label>
              <select 
                name="course" 
                value={formData.course} 
                onChange={handleChange}
                className="w-full bg-[#2a2a2a] text-white px-4 py-3 rounded border border-dark-border focus:outline-none focus:border-blue-500 transition appearance-none"
              >
                <option value="B.Tech">B.Tech</option>
                <option value="BCA">BCA</option>
                <option value="BBA">BBA</option>
                <option value="BA">BA</option>
              </select>
            </div>

            <div>
              <label className="block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wide">Department</label>
              <select 
                name="department" 
                value={formData.department} 
                onChange={handleChange}
                className="w-full bg-[#2a2a2a] text-white px-4 py-3 rounded border border-dark-border focus:outline-none focus:border-blue-500 transition appearance-none"
              >
                <option value="CSE">Computer Science (CSE)</option>
                <option value="ECE">Electronics (ECE)</option>
                <option value="MECH">Mechanical (MECH)</option>
                <option value="BIOTECH">Biotech</option>
                <option value="OTHER">Other</option>
              </select>
            </div>

            <div>
              <label className="block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wide">Course Year</label>
              <select 
                name="course_year" 
                value={formData.course_year} 
                onChange={handleChange}
                className="w-full bg-[#2a2a2a] text-white px-4 py-3 rounded border border-dark-border focus:outline-none focus:border-blue-500 transition appearance-none"
              >
                <option value={1}>Year 1</option>
                <option value={2}>Year 2</option>
                <option value={3}>Year 3</option>
                <option value={4}>Year 4</option>
              </select>
            </div>

            <div>
              <label className="block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wide">Batch</label>
              <input 
                type="text" 
                name="batch"
                value={formData.batch}
                onChange={handleChange}
                className="w-full bg-[#2a2a2a] text-white px-4 py-3 rounded border border-dark-border focus:outline-none focus:border-blue-500 transition" 
                placeholder="e.g., B7"
                required 
              />
            </div>

            <div>
              <label className="block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wide">Section</label>
              <input 
                type="text" 
                name="section"
                value={formData.section}
                onChange={handleChange}
                className="w-full bg-[#2a2a2a] text-white px-4 py-3 rounded border border-dark-border focus:outline-none focus:border-blue-500 transition" 
                placeholder="e.g., S4"
                required 
              />
            </div>

            <div>
              <label className="block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wide">Student Group</label>
              <input 
                type="text" 
                name="student_group"
                value={formData.student_group}
                onChange={handleChange}
                className="w-full bg-[#2a2a2a] text-white px-4 py-3 rounded border border-dark-border focus:outline-none focus:border-blue-500 transition" 
                placeholder="e.g., G2"
                required 
              />
            </div>

          </div>

          <button 
            type="submit" 
            disabled={isLoading}
            className={`w-full bg-blue-600 text-white font-bold py-3 px-4 rounded shadow-lg transition mt-8 tracking-wide ${isLoading ? 'opacity-50 cursor-not-allowed' : 'hover:bg-blue-500'}`}
          >
            {isLoading ? 'Saving Profile...' : 'Enter the Arena'}
          </button>
        </form>

      </div>
    </div>
  );
}