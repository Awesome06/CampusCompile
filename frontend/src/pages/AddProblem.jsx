import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import api from '../services/api'; 
import Button from '../components/ui/Button';

export default function AddProblem() {
  const navigate = useNavigate();
  const [isAuthorized, setIsAuthorized] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const [file, setFile] = useState(null);
  const [status, setStatus] = useState({ type: '', message: '' });
  const [isSubmitting, setIsSubmitting] = useState(false);

  // CHECK PERMISSIONS ON LOAD
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

  const handleFileChange = (e) => {
    const selectedFile = e.target.files[0];
    if (selectedFile && selectedFile.name.endsWith('.zip')) {
      setFile(selectedFile);
      setStatus({ type: '', message: '' });
    } else {
      setFile(null);
      setStatus({ type: 'error', message: 'Please select a valid .zip Polygon package.' });
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!file) {
      setStatus({ type: 'error', message: 'No package selected.' });
      return;
    }

    setIsSubmitting(true);
    setStatus({ type: 'info', message: 'Uploading and extracting Polygon package... 🚀' });

    const formData = new FormData();
    formData.append('package', file);

    try {
      // Direct Multipart upload to the new Polygon endpoint
      const res = await api.post('/problems/import/polygon', formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      });
      
      setStatus({ type: 'success', message: 'Problem synced and forged successfully! 🎉' });
      
      // Redirect to the newly created problem arena
      setTimeout(() => navigate(`/arena/${res.data.problem_id}`), 2000);
    } catch (err) {
      setStatus({ type: 'error', message: err.response?.data?.error || 'Failed to parse Polygon package.' });
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
        <h2 className="text-3xl font-bold text-white mb-2 tracking-wide">Bulk Import (Modality C)</h2>
        <p className="text-gray-400 text-sm mb-6 border-b border-dark-border pb-6">
          Upload a Codeforces Polygon <code className="bg-dark-bg px-1 py-0.5 rounded text-blue-400">.zip</code> package. The system will automatically extract the description, time/memory limits, checker scripts, and test cases.
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

        <form onSubmit={handleSubmit} className="space-y-8">
          
          <div className="flex flex-col items-center justify-center w-full">
            <label className="flex flex-col items-center justify-center w-full h-64 border-2 border-dark-border border-dashed rounded-lg cursor-pointer bg-dark-bg hover:bg-[#2a2a2a] transition">
              <div className="flex flex-col items-center justify-center pt-5 pb-6">
                <svg className="w-10 h-10 mb-4 text-gray-500" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 20 16">
                  <path stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 13h3a3 3 0 0 0 0-6h-.025A5.56 5.56 0 0 0 16 6.5 5.5 5.5 0 0 0 5.207 5.021C5.137 5.017 5.071 5 5 5a4 4 0 0 0 0 8h2.167M10 15V6m0 0L8 8m2-2 2 2"/>
                </svg>
                <p className="mb-2 text-sm text-gray-400">
                  <span className="font-semibold text-blue-500">Click to upload</span> or drag and drop
                </p>
                <p className="text-xs text-gray-500">Polygon Windows/Linux Package (.zip)</p>
              </div>
              <input 
                type="file" 
                className="hidden" 
                accept=".zip"
                onChange={handleFileChange} 
              />
            </label>
          </div>

          {file && (
            <div className="bg-dark-bg p-3 rounded border border-dark-border flex justify-between items-center">
              <span className="text-sm text-gray-300 font-mono">{file.name}</span>
              <span className="text-xs text-gray-500">{(file.size / 1024).toFixed(1)} KB</span>
            </div>
          )}

          <div className="flex justify-end pt-4">
            <Button 
              type="submit" 
              variant="success" 
              disabled={isSubmitting || !file}
            >
              {isSubmitting ? 'Extracting Package...' : 'Sync Problem Data'}
            </Button>
          </div>

        </form>
      </div>
    </div>
  );
}