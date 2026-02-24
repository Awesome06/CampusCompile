import React, { useState } from 'react';
import Editor from '@monaco-editor/react';
import { useParams } from 'react-router-dom';
import axios from 'axios';

export default function Arena() {
  const { id } = useParams();
  
  const [code, setCode] = useState('// Write your C++ code here\n#include <iostream>\n\nint main() {\n    std::cout << "0 1\\n";\n    return 0;\n}');
  const [language, setLanguage] = useState('cpp');
  
  // We now track the raw status string directly (e.g., "Pending", "Accepted", "Wrong Answer")
  const [submitStatus, setSubmitStatus] = useState(''); 

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
      
      // 🚀 Start polling the server for the final verdict!
      pollSubmissionStatus(subId);

    } catch (error) {
      console.error("Submission Error:", error);
      setSubmitStatus('Error: Unauthorized or Server Dead');
    }
  };

  // --- THE NEW POLLING FUNCTION ---
  const pollSubmissionStatus = async (submissionId) => {
  try {
    const res = await axios.get(`http://localhost:8080/api/submissions/${submissionId}`);
    const currentStatus = res.data.status;
    
    console.log("Current Verdict from Backend:", currentStatus); // <-- Debugging line

    // If the status is still "Pending" or "Running", keep polling
    if (currentStatus === 'Pending' || currentStatus === 'Running') {
      setTimeout(() => pollSubmissionStatus(submissionId), 1000);
      setSubmitStatus(currentStatus); // Keep the UI updated with "Running"
    } else {
      // 🎉 SUCCESS: The worker finished! 
      // This will capture "Accepted", "WA", "TLE", etc.
      setSubmitStatus(currentStatus);
    }
  } catch (err) {
    console.error("Polling Error:", err);
    setSubmitStatus('Error fetching status');
  }
};

  // Helper to colorize the output terminal
  const getStatusColor = () => {
    if (submitStatus === 'Accepted') return 'text-green-400 font-bold';
    if (submitStatus.includes('Wrong Answer') || submitStatus.includes('Error')) return 'text-red-400 font-bold';
    if (submitStatus.includes('Pending')) return 'text-yellow-400 animate-pulse';
    return 'text-gray-400';
  };

  return (
    <div className="flex h-[calc(100vh-61px)] w-full"> 
      
      {/* LEFT PANE: Problem Description */}
      <div className="w-1/2 p-6 overflow-y-auto border-r border-dark-border bg-dark-bg">
        <h2 className="text-2xl font-bold mb-4 text-white">Problem ID:</h2>
        <code className="text-xs text-dark-accent bg-dark-surface p-2 rounded mb-6 inline-block">{id}</code>
        <p className="text-gray-300 mb-4 leading-relaxed">
          Solve the problem described here. Once you are ready, hit the Submit Code button to send it to the execution engine.
        </p>
        
        {/* Dynamic System Status Box */}
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
              className="bg-dark-bg text-gray-300 px-3 py-1 rounded border border-dark-border outline-none cursor-pointer"
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