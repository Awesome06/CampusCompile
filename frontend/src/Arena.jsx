import React, { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import { useParams } from 'react-router-dom';
import api from '../services/api'; // 👈 Swapped axios for our custom API service

const boilerplates = {
  cpp: `#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n    // Write your C++ code here\n    return 0;\n}`,
  python: `# Write your Python code here`,
  java: `import java.util.*;\nimport java.io.*;\n\n public class Main {\n    public static void main(String[] args) {\n        // Write your Java code here\n    }\n}`
};

export default function Arena() {
  const { id } = useParams();
  
  const [problem, setProblem] = useState(null);
  const [code, setCode] = useState(boilerplates['cpp']);
  const [language, setLanguage] = useState('cpp');
  const [submitStatus, setSubmitStatus] = useState(''); 
  const [isConsoleOpen, setIsConsoleOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('input'); // 'input' or 'output'
  const [customInput, setCustomInput] = useState('');
  const [consoleOutput, setConsoleOutput] = useState('');
  const [leftTab, setLeftTab] = useState('description'); // 'description' or 'history'
  const [history, setHistory] = useState([]);

  // --- FETCH PROBLEM DATA ON LOAD ---
  useEffect(() => {
    // Look how clean this is! The URL base and Token are handled automatically.
    api.get(`/problems/${id}`)
      .then(res => setProblem(res.data))
      .catch(err => {
        console.error("Could not fetch problem details", err);
      });
      
    fetchHistory();
  }, [id]);

  // Fetch the user's submission history
  const fetchHistory = async () => {
    try {
      const res = await api.get(`/submissions/history/${id}`);
      setHistory(res.data);
    } catch (err) {
      console.error("Could not fetch history:", err);
    }
  };

  const formatText = (text) => {
    if (!text) return "";
    return text.replace(/\\n/g, '\n');
  };

  const handleCopy = (text) => {
    navigator.clipboard.writeText(formatText(text));
  };

  const handleSubmit = async () => {
    setSubmitStatus('Submitting to Go API... 🚀');
    
    try {
      const response = await api.post('/submit', {
        problem_id: id,
        language: language,
        source_code: code
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
      const res = await api.get(`/submissions/${submissionId}`);
      const currentStatus = res.data.status;
      
      console.log("Current Verdict from Backend:", currentStatus); 

      if (currentStatus === 'Pending' || currentStatus === 'Running') {
        setTimeout(() => pollSubmissionStatus(submissionId), 1000);
        setSubmitStatus(currentStatus); 
      } else {
        setSubmitStatus(currentStatus);
        fetchHistory();

        if (currentStatus === 'CE' || currentStatus === 'RE' || currentStatus === 'WA') {
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
    
    try {
      const response = await api.post('/run', {
        language: language,
        source_code: code,
        custom_input: customInput
      });

      pollRunStatus(response.data.run_id);
    } catch (error) {
      setConsoleOutput('Error: Could not connect to execution engine.');
    }
  };

  const pollRunStatus = async (runId) => {
    try {
      const res = await api.get(`/run/${runId}`);
      
      if (res.data.status === 'Pending') {
        setTimeout(() => pollRunStatus(runId), 1000);
      } else {
        setConsoleOutput(res.data.output || "Program finished successfully with no output.");
      }
    } catch (err) {
      setConsoleOutput('Error polling execution status.');
    }
  };

  if (!problem) return <div className="flex justify-center items-center h-screen bg-dark-bg text-white text-xl">Loading Arena...</div>;

  return (
    <div className="flex h-[calc(100vh-61px)] w-full"> 
      
      {/* LEFT PANE: Dynamic Content (Tabs) */}
      <div className="w-1/2 flex flex-col border-r border-dark-border bg-dark-bg">
        
        {/* Left Pane Tab Bar */}
        <div className="flex items-center px-4 bg-[#1e1e1e] border-b border-dark-border select-none">
          <button 
            className={`py-3 px-4 text-sm font-bold tracking-wide transition ${leftTab === 'description' ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`}
            onClick={() => setLeftTab('description')}
          >
            Description
          </button>
          <button 
            className={`py-3 px-4 text-sm font-bold tracking-wide transition ${leftTab === 'history' ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`}
            onClick={() => setLeftTab('history')}
          >
            Submissions
          </button>
        </div>

        {/* Tab Content Area */}
        <div className="flex-grow p-6 overflow-y-auto">
          {leftTab === 'history' && (
            <div>
              <h3 className="text-xl font-bold text-white mb-6">Submission History</h3>
              
              <div className="bg-[#1e1e1e] border border-dark-border rounded-lg overflow-hidden shadow-inner">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="bg-[#2a2a2a] border-b border-dark-border text-gray-400 text-xs uppercase tracking-wider">
                      <th className="p-4 font-semibold">Time Submitted</th>
                      <th className="p-4 font-semibold">Status</th>
                      <th className="p-4 font-semibold">Language</th>
                    </tr>
                  </thead>
                  <tbody>
                    {history.length === 0 ? (
                      <tr>
                        <td colSpan="3" className="p-6 text-center text-gray-500 italic">
                          No submissions yet. Step into the arena!
                        </td>
                      </tr>
                    ) : (
                      history.map(sub => (
                        <tr key={sub.submission_id} className="border-b border-dark-border last:border-0 hover:bg-[#2a2a2a] transition">
                          <td className="p-4 text-sm text-gray-300">
                            {new Date(sub.submitted_at).toLocaleString([], { dateStyle: 'short', timeStyle: 'short' })}
                          </td>
                          <td className={`p-4 text-sm font-bold ${sub.status === 'AC' || sub.status === 'Accepted' ? 'text-green-400' : 'text-red-400'}`}>
                            {sub.status}
                          </td>
                          <td className="p-4 text-sm text-gray-300 uppercase">
                            {sub.language}
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
          {leftTab === 'description' && (
            <>
              <h2 className="text-3xl font-bold mb-4 text-white">{problem.title}</h2>
              
              {/* Dynamic Difficulty Badge */}
              <span className={`px-2 py-1 text-xs rounded font-bold mb-6 inline-block shadow-sm ${
                problem.difficulty === 'Easy' ? 'bg-green-900/50 text-green-400 border border-green-800' : 
                problem.difficulty === 'Medium' ? 'bg-yellow-900/50 text-yellow-400 border border-yellow-800' : 
                'bg-red-900/50 text-red-400 border border-red-800'
              }`}>
                {problem.difficulty}
              </span>
              
              {/* Problem Description */}
              <div className="text-gray-300 mb-8 leading-relaxed whitespace-pre-wrap">
                {problem.description}
              </div>
              
              {/* Sample Test Case UI */}
              {(problem.sample_input != null && problem.sample_output != null) && (
                <div className="mb-8">
                  <h3 className="text-lg font-bold text-white mb-3 tracking-wide">Sample Test Case</h3>
                  <div className="bg-[#1e1e1e] border border-dark-border rounded-lg overflow-hidden shadow-inner">
                    <div className="p-4 border-b border-dark-border relative group">
                      <div className="flex justify-between items-center mb-2">
                        <span className="text-xs font-bold text-gray-500 uppercase tracking-wider block">Input:</span>
                        <button onClick={() => handleCopy(problem.sample_input)} className="text-xs text-gray-400 hover:text-white bg-dark-bg px-2 py-1 rounded border border-dark-border opacity-0 group-hover:opacity-100 transition absolute top-2 right-2 shadow">Copy</button>
                      </div>
                      <pre className="font-mono text-gray-300 whitespace-pre-wrap">{formatText(problem.sample_input)}</pre>
                    </div>
                    <div className="p-4 bg-[#1a1a1a] relative group">
                      <div className="flex justify-between items-center mb-2">
                        <span className="text-xs font-bold text-gray-500 uppercase tracking-wider block">Expected Output:</span>
                        <button onClick={() => handleCopy(problem.sample_output)} className="text-xs text-gray-400 hover:text-white bg-dark-bg px-2 py-1 rounded border border-dark-border opacity-0 group-hover:opacity-100 transition absolute top-2 right-2 shadow">Copy</button>
                      </div>
                      <pre className="font-mono text-gray-300 whitespace-pre-wrap">{formatText(problem.sample_output)}</pre>
                    </div>
                  </div>
                </div>
              )}

              {/* Verdict Box */}
              <div className="mt-8 p-4 bg-[#2a2a2a] border border-dark-border rounded shadow-inner">
                <h3 className="text-sm font-bold text-gray-400 mb-2">LATEST VERDICT:</h3>
                <p className={`font-mono text-lg tracking-wide ${getStatusColor()}`}>
                  {submitStatus || "Awaiting submission..."}
                </p>
              </div>
            </>
          )}

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
        
        {/* Monaco Editor */}
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

        {/* BOTTOM CONSOLE */}
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