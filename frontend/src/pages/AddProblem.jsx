import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import api from '../services/api'; 
import Button from '../components/ui/Button';

export default function AddProblem() {
  const navigate = useNavigate();
  const [isAuthorized, setIsAuthorized] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  // Wizard State
  const [step, setStep] = useState(1);
  const [createdProblemId, setCreatedProblemId] = useState(null);
  const [status, setStatus] = useState({ type: '', message: '' });
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Step 1: Problem Details
  const [problemData, setProblemData] = useState({
    title: '',
    description: '',
    difficulty: 'Easy',
    time_limit: 2000,
    memory_limit: 256000
  });

  // Step 2: S3 Test Cases
  const [inputFile, setInputFile] = useState(null);
  const [expectedFile, setExpectedFile] = useState(null);

  // CHECK PERMISSIONS ON LOAD (Admins & Professors allowed)
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

  // Handle Step 1 Submission
  const handleCreateProblem = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);
    setStatus({ type: 'info', message: 'Forging problem in database... 🚀' });

    try {
      const res = await api.post('/problems', problemData);
      setCreatedProblemId(res.data.problem_id);
      setStatus({ type: 'success', message: 'Problem created! Now, upload the test cases.' });
      setStep(2); // Move to S3 upload step
    } catch (err) {
      setStatus({ type: 'error', message: err.response?.data?.error || 'Failed to create problem.' });
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle Step 2 Submission (S3 MinIO)
  const handleUploadTestCases = async (e) => {
    e.preventDefault();
    if (!inputFile || !expectedFile) {
      setStatus({ type: 'error', message: 'Please select both input and expected output files.' });
      return;
    }

    setIsSubmitting(true);
    setStatus({ type: 'info', message: 'Streaming test cases to MinIO (S3)... ⏳' });

    const formData = new FormData();
    formData.append('problem_id', createdProblemId);
    formData.append('input_file', inputFile);
    formData.append('expected_file', expectedFile);

    try {
      await api.post('/problems/testcases', formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      });
      
      setStatus({ type: 'success', message: 'Test cases securely streamed! Redirecting to Arena... 🎉' });
      setTimeout(() => navigate(`/arena/${createdProblemId}`), 2000);
    } catch (err) {
      setStatus({ type: 'error', message: err.response?.data?.error || 'Failed to upload test cases.' });
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
    <div className="min-h-[calc(100vh-61px)] bg-dark-bg text-gray-300 flex justify-center py-10 px-4">
      <div className="w-full max-w-2xl bg-[#1e1e1e] p-8 rounded-lg shadow-xl border border-dark-border h-max">
        <h2 className="text-3xl font-bold text-white mb-2 tracking-wide">
          {step === 1 ? 'Forge New Problem' : 'Upload Test Cases (S3)'}
        </h2>
        <p className="text-gray-400 text-sm mb-6 border-b border-dark-border pb-6">
          {step === 1 
            ? 'Define the constraints and description for the new competitive programming challenge.'
            : 'Securely stream massive .txt test cases directly to your MinIO object storage.'}
        </p>

        {status.message && (
          <div className={`p-4 mb-6 rounded font-bold text-sm ${
            status.type === 'error' ? 'bg-red-900/50 text-red-400 border border-red-800' :
            status.type === 'success' ? 'bg-green-900/50 text-green-400 border border-green-800' :
            'bg-blue-900/50 text-blue-400 border border-blue-800'
          }`}>
            {status.message}
          </div>
        )}

        {/* --- STEP 1: PROBLEM DETAILS --- */}
        {step === 1 && (
          <form onSubmit={handleCreateProblem} className="space-y-4">
            <div>
              <label className="text-xs font-bold text-gray-400 uppercase">Title</label>
              <input required type="text" className="w-full bg-dark-bg text-white p-2 rounded border border-dark-border mt-1" 
                value={problemData.title} onChange={e => setProblemData({...problemData, title: e.target.value})} />
            </div>
            <div>
              <label className="text-xs font-bold text-gray-400 uppercase">Description (LaTeX Supported)</label>
              <textarea required rows="6" className="w-full bg-dark-bg text-white p-2 rounded border border-dark-border mt-1 font-mono text-sm"
                value={problemData.description} onChange={e => setProblemData({...problemData, description: e.target.value})} />
            </div>
            <div className="flex space-x-4">
              <div className="w-1/3">
                <label className="text-xs font-bold text-gray-400 uppercase">Difficulty</label>
                <select className="w-full bg-dark-bg text-white p-2 rounded border border-dark-border mt-1"
                  value={problemData.difficulty} onChange={e => setProblemData({...problemData, difficulty: e.target.value})}>
                  <option>Easy</option><option>Medium</option><option>Hard</option>
                </select>
              </div>
              <div className="w-1/3">
                <label className="text-xs font-bold text-gray-400 uppercase">Time Limit (ms)</label>
                <input required type="number" className="w-full bg-dark-bg text-white p-2 rounded border border-dark-border mt-1"
                  value={problemData.time_limit} onChange={e => setProblemData({...problemData, time_limit: parseInt(e.target.value)})} />
              </div>
              <div className="w-1/3">
                <label className="text-xs font-bold text-gray-400 uppercase">Memory Limit (KB)</label>
                <input required type="number" className="w-full bg-dark-bg text-white p-2 rounded border border-dark-border mt-1"
                  value={problemData.memory_limit} onChange={e => setProblemData({...problemData, memory_limit: parseInt(e.target.value)})} />
              </div>
            </div>
            <div className="flex justify-end pt-4">
              <Button type="submit" variant="primary" disabled={isSubmitting}>
                {isSubmitting ? 'Creating...' : 'Next: Upload Test Cases'}
              </Button>
            </div>
          </form>
        )}

        {/* --- STEP 2: S3 UPLOAD --- */}
        {step === 2 && (
          <form onSubmit={handleUploadTestCases} className="space-y-6">
            <div className="flex flex-col space-y-2">
              <label className="text-xs font-bold text-gray-400 uppercase tracking-widest">Input File (.txt)</label>
              <input type="file" accept=".txt" onChange={(e) => setInputFile(e.target.files[0])}
                className="block w-full text-sm text-gray-400 file:mr-4 file:py-2 file:px-4 file:rounded file:border-0 file:text-sm file:font-bold file:bg-dark-accent file:text-white hover:file:bg-blue-600 transition" />
            </div>

            <div className="flex flex-col space-y-2">
              <label className="text-xs font-bold text-gray-400 uppercase tracking-widest">Expected Output File (.txt)</label>
              <input type="file" accept=".txt" onChange={(e) => setExpectedFile(e.target.files[0])}
                className="block w-full text-sm text-gray-400 file:mr-4 file:py-2 file:px-4 file:rounded file:border-0 file:text-sm file:font-bold file:bg-green-700 file:text-white hover:file:bg-green-600 transition" />
            </div>

            <div className="flex justify-end pt-4 border-t border-dark-border mt-4">
              <Button type="submit" variant="success" disabled={isSubmitting || !inputFile || !expectedFile}>
                {isSubmitting ? 'Streaming to S3...' : 'Finalize & Publish'}
              </Button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}