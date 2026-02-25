import React, { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import { useParams } from 'react-router-dom';
import axios from 'axios';

const boilerplates = {
  cpp: `#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n    // Write your C++ code here\n    return 0;\n}`,
  python: `# Write your Python code here`,
  java: `import java.util.*;\nimport java.io.*;\n\n public class Main {\n    public static void main(String[] args) {\n        // Write your Java code here\n    }\n}`
};

export default function Arena() {
  const { id } = useParams();
  
  // 👇 New state to hold the fetched problem data
  const [problem, setProblem] = useState(null);
  const [code, setCode] = useState(boilerplates['cpp']);
  const [language, setLanguage] = useState('cpp');
  const [submitStatus, setSubmitStatus] = useState(''); 

  const [isConsoleOpen, setIsConsoleOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('input'); // 'input' or 'output'
  const [customInput, setCustomInput] = useState('');
  const [consoleOutput, setConsoleOutput] = useState('');

 // --- NEW: FETCH PROBLEM DATA ON LOAD (PROTECTED) ---
  useEffect(() => {
    const token = localStorage.getItem('token'); // Grab the token

    axios.get(`http://localhost:8080/api/problems/${id}`, {
      headers: {
        'Authorization': `Bearer ${token}` // Attach the VIP pass
      }
    })
      .then(res => setProblem(res.data))
      .catch(err => {
        console.error("Could not fetch problem details", err);
        // Optional: If unauthorized, you could kick them back to login here
        if (err.response && err.response.status === 401) {
          window.location.href = '/login'; 
        }
      });
  }, [id]);

  const handleSubmit = async () => {
    setSubmitStatus('Submitting to Go API... 🚀');

    // 👇 YOUR REAL JWT TOKEN
    const token = localStorage.getItem('token');

    if (!token) {
      setSubmitStatus('Error: You must be logged in to submit code.');
      return;
    }
    
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
    const token = localStorage.getItem('token'); // Grab the token

    try {
      const res = await axios.get(`http://localhost:8080/api/submissions/${submissionId}`, {
        headers: {
          'Authorization': `Bearer ${token}` // Attach the VIP pass
        }
      });
      const currentStatus = res.data.status;
      
      console.log("Current Verdict from Backend:", currentStatus); 

      if (currentStatus === 'Pending' || currentStatus === 'Running') {
        setTimeout(() => pollSubmissionStatus(submissionId), 1000);
        setSubmitStatus(currentStatus); 
      } else {
        setSubmitStatus(currentStatus);

        if (currentStatus === 'CE' || currentStatus === 'RE' || currentStatus === 'WA') {
          // Note: res.data.message will require a small backend update (explained below)
          setConsoleOutput(res.data.message || `Verdict: ${currentStatus}\nNo detailed logs provided by server.`);
          setActiveTab('output');
          setIsConsoleOpen(true);
        } else if (currentStatus === 'AC' || currentStatus === 'Accepted') {
          setConsoleOutput("Execution Successful! 🎉\nAll test cases passed.");
          setActiveTab('output');
          setIsConsoleOpen(true);
        }
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

  const handleRunCode = async () => {
    setConsoleOutput('Spinning up sandbox... ⚙️');
    setActiveTab('output');
    setIsConsoleOpen(true);

    const token = localStorage.getItem('token');
    
    try {
      const response = await axios.post('http://localhost:8080/api/run', {
        language: language,
        source_code: code,
        custom_input: customInput
      }, {
        headers: { Authorization: `Bearer ${token}` }
      });

      pollRunStatus(response.data.run_id);
    } catch (error) {
      setConsoleOutput('Error: Could not connect to execution engine.');
    }
  };

  const pollRunStatus = async (runId) => {
    const token = localStorage.getItem('token');
    try {
      const res = await axios.get(`http://localhost:8080/api/run/${runId}`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      
      if (res.data.status === 'Pending') {
        setTimeout(() => pollRunStatus(runId), 1000);
      } else {
        // Output the raw execution logs
        setConsoleOutput(res.data.output || "Program finished successfully with no output.");
      }
    } catch (err) {
      setConsoleOutput('Error polling execution status.');
    }
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

      {/* RIGHT PANE: Code Editor & Console */}
      <div className="w-1/2 flex flex-col bg-dark-surface border-l border-dark-border">
        
        {/* Editor Header */}
        <div className="flex justify-between items-center p-2 bg-[#1e1e1e] border-b border-dark-border">
            <select 
              value={language}
              onChange={(e) => {
                const newLang = e.target.value;
                setLanguage(newLang);
                setCode(boilerplates[newLang]);
              }}
              className="bg-dark-bg text-gray-300 px-3 py-1 rounded border border-dark-border outline-none cursor-pointer hover:border-gray-500 transition"
            >
                <option value="cpp">C++</option>
                <option value="python">Python</option>
                <option value="java">Java</option>
            </select>
            
            {/* Wrap both buttons in a flex container to align them on the right */}
            <div className="flex space-x-3">
                <button 
                  onClick={handleRunCode}
                  className="bg-gray-700 text-white px-5 py-1.5 rounded font-bold hover:bg-gray-600 transition shadow-lg"
                >
                    Run Code
                </button>
                <button 
                  onClick={handleSubmit}
                  className="bg-dark-success text-white px-5 py-1.5 rounded font-bold hover:bg-green-600 transition shadow-lg"
                >
                    Submit Code
                </button>
            </div>
        </div>
        
        {/* Monaco Editor (Takes remaining space above console) */}
        <div className="flex-grow overflow-hidden relative">
          <Editor
            height="100%"
            language={language}
            theme="vs-dark"
            value={code}
            onChange={(value) => setCode(value)}
            options={{ minimap: { enabled: false }, fontSize: 16, wordWrap: 'on', padding: { top: 16 } }}
          />
        </div>

        {/* 👇 THE NEW BOTTOM CONSOLE 👇 */}
        <div className={`flex flex-col border-t border-dark-border bg-[#1e1e1e] transition-all duration-300 ease-in-out ${isConsoleOpen ? 'h-64' : 'h-10'}`}>
            
            {/* Console Tab Bar (Clickable) */}
            <div className="flex items-center justify-between px-4 py-2 bg-dark-surface cursor-pointer select-none" onClick={() => setIsConsoleOpen(!isConsoleOpen)}>
                <div className="flex space-x-6">
                    <button 
                      className={`text-sm font-bold tracking-wide transition ${activeTab === 'input' && isConsoleOpen ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`}
                      onClick={(e) => { e.stopPropagation(); setActiveTab('input'); setIsConsoleOpen(true); }}
                    >
                        Custom Input
                    </button>
                    <button 
                      className={`text-sm font-bold tracking-wide transition ${activeTab === 'output' && isConsoleOpen ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`}
                      onClick={(e) => { e.stopPropagation(); setActiveTab('output'); setIsConsoleOpen(true); }}
                    >
                        Output / Errors
                    </button>
                </div>
                <span className="text-gray-400 text-xs font-bold uppercase tracking-wider hover:text-white transition">
                    {isConsoleOpen ? '▼ Close' : '▲ Console'}
                </span>
            </div>

            {/* Console Content Area */}
            {isConsoleOpen && (
                <div className="flex-grow p-4 bg-dark-bg overflow-hidden">
                    {activeTab === 'input' ? (
                        <textarea 
                            className="w-full h-full bg-[#1e1e1e] text-gray-300 p-3 rounded border border-dark-border outline-none resize-none font-mono text-sm focus:border-gray-500 transition"
                            placeholder="Enter your custom input here..."
                            value={customInput}
                            onChange={(e) => setCustomInput(e.target.value)}
                        />
                    ) : (
                        <div className="w-full h-full bg-[#1e1e1e] text-red-300 p-3 rounded border border-dark-border overflow-y-auto font-mono text-sm whitespace-pre-wrap">
                            {consoleOutput || "Run code to see output..."}
                        </div>
                    )}
                </div>
            )}
        </div>
      </div>
    </div>
  );
}