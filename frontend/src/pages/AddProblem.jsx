import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import api from '../services/api'; 

// 👇 Import your shiny new UI components!
import Input from '../components/ui/Input';
import Select from '../components/ui/Select';
import TextArea from '../components/ui/TextArea';

export default function AddProblem() {
  const navigate = useNavigate();
  const [isAuthorized, setIsAuthorized] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const [formData, setFormData] = useState({
    title: '',
    description: '',
    difficulty: 'Easy',
    sample_input: '',
    sample_output: '',
    time_limit: 1.0,
    memory_limit: 256 
  });

  const [status, setStatus] = useState({ type: '', message: '' });
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) {
      navigate('/login');
      return;
    }

    try {
      const decodedToken = jwtDecode(token);
      const userRole = decodedToken.role?.toLowerCase(); 
      setIsAuthorized(userRole === 'admin' || userRole === 'professor');
    } catch (error) {
      setIsAuthorized(false);
    } finally {
      setIsLoading(false);
    }
  }, [navigate]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);
    setStatus({ type: 'info', message: 'Adding problem to the arena... 🚀' });

    try {
      const payload = {
        ...formData,
        time_limit: parseFloat(formData.time_limit),
        memory_limit: parseInt(formData.memory_limit, 10)
      };

      const res = await api.post('/problems', payload);
      setStatus({ type: 'success', message: 'Problem added successfully! 🎉' });
      
      setTimeout(() => navigate(`/arena/${res.data.problem_id}`), 2000);
    } catch (err) {
      setStatus({ type: 'error', message: err.response?.data?.error || 'Failed to add problem.' });
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoading) return <div className="flex justify-center items-center h-[calc(100vh-61px)] bg-dark-bg text-white text-xl">Verifying permissions...</div>;

  if (!isAuthorized) {
    return (
      <div className="flex justify-center items-center h-[calc(100vh-61px)] bg-dark-bg">
        <div className="text-center p-8 bg-[#1e1e1e] border border-red-800 rounded-lg shadow-xl">
          <h2 className="text-3xl font-bold text-red-500 mb-4">Access Denied 🛑</h2>
          <p className="text-gray-300 text-lg">Only Admins and Professors have clearance to forge new problems.</p>
          <button onClick={() => navigate('/')} className="mt-6 bg-gray-700 text-white px-6 py-2 rounded font-bold hover:bg-gray-600 transition">
            Return to Safety
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-[calc(100vh-61px)] bg-dark-bg text-gray-300 flex justify-center py-10">
      <div className="w-full max-w-4xl bg-[#1e1e1e] p-8 rounded-lg shadow-xl border border-dark-border">
        <h2 className="text-3xl font-bold text-white mb-6 border-b border-dark-border pb-4">Create New Problem</h2>

        {status.message && (
          <div className={`p-4 mb-6 rounded font-bold ${
            status.type === 'error' ? 'bg-red-900/50 text-red-400 border border-red-800' :
            status.type === 'success' ? 'bg-green-900/50 text-green-400 border border-green-800' :
            'bg-blue-900/50 text-blue-400 border border-blue-800'
          }`}>
            {status.message}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          
          <div className="flex gap-6">
            <div className="flex-grow">
              <Input label="Problem Title" name="title" value={formData.title} onChange={handleChange} placeholder="e.g., Two Sum" required />
            </div>
            <div className="w-1/3">
              <Select 
                label="Difficulty" name="difficulty" value={formData.difficulty} onChange={handleChange} 
                options={[{value: 'Easy', label: 'Easy'}, {value: 'Medium', label: 'Medium'}, {value: 'Hard', label: 'Hard'}]} 
              />
            </div>
          </div>

          <div className="flex gap-6">
            <div className="w-1/2">
              <Input label="Time Limit (Seconds)" name="time_limit" type="number" step="0.1" value={formData.time_limit} onChange={handleChange} required />
            </div>
            <div className="w-1/2">
              <Input label="Memory Limit (MB)" name="memory_limit" type="number" value={formData.memory_limit} onChange={handleChange} required />
            </div>
          </div>

          <TextArea label="Problem Description" name="description" value={formData.description} onChange={handleChange} placeholder="Explain the problem clearly here..." rows={6} required />

          <div className="flex gap-6">
            <div className="w-1/2">
              <TextArea label="Sample Input" name="sample_input" value={formData.sample_input} onChange={handleChange} placeholder="2 7 11 15\n9" required />
            </div>
            <div className="w-1/2">
              <TextArea label="Sample Output" name="sample_output" value={formData.sample_output} onChange={handleChange} placeholder="0 1" required />
            </div>
          </div>

          <div className="flex justify-end mt-8 border-t border-dark-border pt-6">
            <button type="submit" disabled={isSubmitting} className={`bg-dark-success text-white px-8 py-3 rounded font-bold shadow-lg transition tracking-wide ${isSubmitting ? 'opacity-50 cursor-not-allowed' : 'hover:bg-green-600'}`}>
              {isSubmitting ? 'Publishing...' : 'Publish Problem'}
            </button>
          </div>

        </form>
      </div>
    </div>
  );
}