import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../services/api';

// 👇 Import your UI Toolkit
import Input from '../components/ui/Input';
import Select from '../components/ui/Select';
import Button from '../components/ui/Button';

export default function Onboarding() {
  const navigate = useNavigate();
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const [formData, setFormData] = useState({
    username: '', course: 'B.Tech', department: 'CSE', 
    course_year: 1, batch: '', section: '', student_group: ''
  });

  useEffect(() => {
    if (!localStorage.getItem('token')) navigate('/login');
  }, [navigate]);

  const handleChange = (e) => {
    setFormData((prev) => ({ ...prev, [e.target.name]: e.target.value }));
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
      setError(err.response?.data?.error || 'Failed to save profile. Please try again.');
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
        
        {error && <div className="bg-red-900/50 border border-red-500 text-red-400 p-3 rounded mb-6 text-sm text-center font-bold">{error}</div>}

        <form onSubmit={handleSubmit} className="space-y-6">
          
          <div>
            <Input label="Arena Username" name="username" value={formData.username} onChange={handleChange} placeholder="e.g., CodeNinja99" required />
            <p className="text-xs text-gray-500 mt-1">This will be your public handle on the leaderboards.</p>
          </div>

          <div className="border-t border-dark-border my-6"></div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Select 
              label="Course" name="course" value={formData.course} onChange={handleChange}
              options={[{value: 'B.Tech', label: 'B.Tech'}, {value: 'BCA', label: 'BCA'}, {value: 'BBA', label: 'BBA'}, {value: 'BA', label: 'BA'}]}
            />
            
            <Select 
              label="Department" name="department" value={formData.department} onChange={handleChange}
              options={[{value: 'CSE', label: 'Computer Science (CSE)'}, {value: 'ECE', label: 'Electronics (ECE)'}, {value: 'MECH', label: 'Mechanical (MECH)'}, {value: 'BIOTECH', label: 'Biotech'}, {value: 'OTHER', label: 'Other'}]}
            />

            <Select 
              label="Course Year" name="course_year" value={formData.course_year} onChange={handleChange}
              options={[{value: 1, label: 'Year 1'}, {value: 2, label: 'Year 2'}, {value: 3, label: 'Year 3'}, {value: 4, label: 'Year 4'}]}
            />

            <Input label="Batch" name="batch" value={formData.batch} onChange={handleChange} placeholder="e.g., B7" required />
            <Input label="Section" name="section" value={formData.section} onChange={handleChange} placeholder="e.g., S4" required />
            <Input label="Student Group" name="student_group" value={formData.student_group} onChange={handleChange} placeholder="e.g., G2" required />
          </div>

          <div className="mt-8">
            <Button type="submit" variant="primary" disabled={isLoading}>
              {isLoading ? 'Saving Profile...' : 'Enter the Arena'}
            </Button>
          </div>
        </form>

      </div>
    </div>
  );
}