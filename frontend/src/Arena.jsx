import React, { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import { useParams } from 'react-router-dom';
import axios from 'axios';

export default function Arena() {
  const { id } = useParams();
  
  // 👇 New state to hold the fetched problem data
  const [problem, setProblem] = useState(null);
  
  const [code, setCode] = useState('// Write your solution here...');
  const [language, setLanguage] = useState('cpp');
  const [submitStatus, setSubmitStatus] = useState(''); 

  // --- NEW: FETCH PROBLEM DATA ON LOAD ---
  useEffect(() => {
    axios.get(`http://localhost:8080/api/problems/${id}`)
      .then(res => setProblem(res.data))
      .catch(err => console.error("Could not fetch problem details", err));
  }, [id]); // Re-run this if the 'id' in the URL changes

  const handleSubmit = async () => {
    setSubmitStatus('Submitting to Go API... 🚀');

    // 👇 YOUR REAL JWT TOKEN
    const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzIxNjg1MzYsInVzZXJfaWQiOiI4MmJjNzZiNC04NzU5LTQxZDYtYjJlNC1hZTdkMDYyODE2NTkifQ.2vVv_rvnxlniCcBf0qL7tx8jX_33fA7hdbjq_fel_6E";

    try {
      const response = await axios.post('http://localhost:8080/api/submit', {
        problem_id: id,
        language: language,
        source_code: code
      }, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        }
      });

      const subId = response.data.submission_id;
      setSubmitStatus('Pending... ⏳');
      
      pollSubmissionStatus(subId);

    } catch (error) {
      console.error("Submission Error:", error);
      setSubmitStatus('Error: Unauthorized or Server Dead');
    }
  };

  const pollSubmissionStatus = async (submissionId) => {
    try {
      const res = await axios.get(`http://localhost:8080/api/submissions/${submissionId}`);
      const currentStatus = res.data.status;
      
      console.log("Current Verdict from Backend:", currentStatus); 

      if (currentStatus === 'Pending' || currentStatus === 'Running') {
        setTimeout(() => pollSubmissionStatus(submissionId), 1000);
        setSubmitStatus(currentStatus); 
      } else {
        setSubmitStatus(currentStatus);
      }
    } catch (err) {
      console.error("Polling Error:", err);
      setSubmitStatus('Error fetching status');
    }
  };

  const getStatusColor = () => {
    if (submitStatus === 'Accepted' || submitStatus === 'AC') return 'text-green-400 font-bold';
    if (submitStatus === 'WA' || submitStatus.includes('Wrong') || submitStatus.includes('Error')) return 'text-red-400 font-bold';
    if (submitStatus === 'Pending' || submitStatus === 'Running') return 'text-yellow-400 animate-pulse';
    return 'text-gray-400';
  };

  // 👇 Wait to render the UI until the Go API returns the problem data
  if (!problem) return <div className="flex justify-center items-center h-screen bg-dark-bg text-white text-xl">Loading Arena...</div>;

  return (
    <div className="flex h-[calc(100vh-61px)] w-full"> 
      
      {/* LEFT PANE: Dynamic Problem Description */}
      <div className="w-1/2 p-6 overflow-y-auto border-r border-dark-border bg-dark-bg">
        <h2 className="text-3xl font-bold mb-4 text-white">{problem.title}</h2>
        
        {/* Dynamic Difficulty Badge */}
        <span className={`px-2 py-1 text-xs rounded font-bold mb-6 inline-block shadow-sm ${
          problem.difficulty === 'Easy' ? 'bg-green-900/50 text-green-400 border border-green-800' : 
          problem.difficulty === 'Medium' ? 'bg-yellow-900/50 text-yellow-400 border border-yellow-800' : 
          'bg-red-900/50 text-red-400 border border-red-800'
        }`}>
          {problem.difficulty}
        </span>
        
        {/* Render the actual database description. whitespace-pre-wrap preserves formatting/newlines */}
        <div className="text-gray-300 mb-4 leading-relaxed whitespace-pre-wrap">
          {problem.description}
        </div>
        
        <div className="mt-8 p-4 bg-[#2a2a2a] border border-dark-border rounded shadow-inner">
          <h3 className="text-sm font-bold text-gray-400 mb-2">VERDICT:</h3>
          <p className={`font-mono text-lg tracking-wide ${getStatusColor()}`}>
            {submitStatus || "Awaiting submission..."}
          </p>
        </div>
      </div>

      {/* RIGHT PANE: Code Editor */}
      <div className="w-1/2 flex flex-col bg-dark-surface">
        <div className="flex justify-between items-center p-2 bg-[#1e1e1e] border-b border-dark-border">
            <select 
              value={language}
              onChange={(e) => setLanguage(e.target.value)}
              className="bg-dark-bg text-gray-300 px-3 py-1 rounded border border-dark-border outline-none cursor-pointer hover:border-gray-500 transition"
            >
                <option value="cpp">C++</option>
                <option value="python">Python</option>
                <option value="java">Java</option>
            </select>
            
            <button 
              onClick={handleSubmit}
              className="bg-dark-success text-white px-5 py-1.5 rounded font-bold hover:bg-green-600 transition shadow-lg"
            >
                Submit Code
            </button>
        </div>
        
        <div className="flex-grow">
          <Editor
            height="100%"
            language={language}
            theme="vs-dark"
            value={code}
            onChange={(value) => setCode(value)}
            options={{ minimap: { enabled: false }, fontSize: 16, wordWrap: 'on', padding: { top: 16 } }}
          />
        </div>
      </div>
    </div>
  );
}