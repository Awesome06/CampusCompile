import React, { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import { useParams, useNavigate } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import remarkGfm from 'remark-gfm';
import ReactMarkdown from 'react-markdown';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import api from '../services/api';
import Button from '../components/ui/Button';

const boilerplates = {
  cpp: `#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n    // Write your C++ code here\n    return 0;\n}`,
  python: `# Write your Python code here`,
  java: `import java.util.*;\nimport java.io.*;\n\n public class Main {\n    public static void main(String[] args) {\n        // Write your Java code here\n    }\n}`
};

export default function Arena() {
  const { id } = useParams();
  const navigate = useNavigate();
  
  const [problem, setProblem] = useState(null);
  const [currentUser, setCurrentUser] = useState(null);
  const [code, setCode] = useState(boilerplates['cpp']);
  const [language, setLanguage] = useState('cpp');
  const [submitStatus, setSubmitStatus] = useState(''); 
  const [isConsoleOpen, setIsConsoleOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('input');
  const [customInput, setCustomInput] = useState('');
  const [consoleOutput, setConsoleOutput] = useState('');
  const [leftTab, setLeftTab] = useState('description');
  const [history, setHistory] = useState([]);
  
  // Modal States
  const [selectedSubmission, setSelectedSubmission] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  useEffect(() => {
    // 1. Fetch Problem Details
    api.get(`/problems/${id}`)
      .then(res => setProblem(res.data))
      .catch(err => console.error("Could not fetch problem details", err));
      
    fetchHistory();

    // 2. Decode Token for RBAC 🛡️
    const token = localStorage.getItem('token');
    if (token) {
      try {
        const decoded = jwtDecode(token);
        setCurrentUser({
          id: decoded.user_id || decoded.sub || decoded.id,
          role: decoded.role?.toLowerCase()
        });
      } catch (err) {
        console.error("Failed to decode token:", err);
      }
    }
  }, [id]);

  const canEdit = currentUser && problem && (currentUser.role === 'admin' || currentUser.id === problem.author_id);

  const fetchHistory = async () => {
    try {
      const res = await api.get(`/submissions/history/${id}`);
      setHistory(res.data || []);
    } catch (err) {
      console.error("Could not fetch history:", err);
    }
  };

  const handleSubmit = async () => {
    setSubmitStatus('Pending... ⏳');
    setIsConsoleOpen(false);
    
    try {
      const response = await api.post('/submit', {
        problem_id: id,
        language: language,
        source_code: code
      });

      pollSubmissionStatus(response.data.submission_id);
    } catch (error) {
      setSubmitStatus('Error: Submission Failed');
    }
  };

  const pollSubmissionStatus = async (submissionId) => {
    try {
      const res = await api.get(`/submissions/${submissionId}`);
      const { status, message } = res.data;
      
      if (status === 'Pending' || status === 'Running') {
        setSubmitStatus(status === 'Running' ? 'Running... ⚙️' : 'Pending... ⏳');
        setTimeout(() => pollSubmissionStatus(submissionId), 1000);
      } else {
        setSubmitStatus(status);
        fetchHistory();

        if (['CE', 'RE', 'WA', 'TLE', 'SE'].includes(status)) {
          setConsoleOutput(message || `Verdict: ${status}`);
          setActiveTab('output');
          setIsConsoleOpen(true);
        } else if (status === 'AC' || status === 'Accepted') {
          setConsoleOutput("Execution Successful! 🎉\nAll test cases passed.");
          setActiveTab('output');
          setIsConsoleOpen(true);
        }
      }
    } catch (err) {
      setSubmitStatus('Error fetching status');
    }
  };

  const handleRunCode = async () => {
    setConsoleOutput('Queuing... ⚙️');
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
        setConsoleOutput(res.data.output || res.data.message || "Program finished with no output.");
      }
    } catch (err) {
      setConsoleOutput('Error polling execution status.');
    }
  };

  const getStatusColor = () => {
    if (['Accepted', 'AC'].includes(submitStatus)) return 'text-green-400 font-bold';
    if (['WA', 'CE', 'RE', 'TLE', 'SE'].includes(submitStatus) || submitStatus.includes('Error')) return 'text-red-400 font-bold';
    if (['Pending', 'Running'].includes(submitStatus) || submitStatus.includes('⏳')) return 'text-yellow-400 animate-pulse';
    return 'text-gray-400';
  };

  const handleViewSubmission = async (submissionId) => {
    try {
      const res = await api.get(`/submissions/${submissionId}`);
      setSelectedSubmission(res.data);
      setIsModalOpen(true);
    } catch (err) {
      console.error("Error fetching submission details:", err);
    }
  };

  if (!problem) return <div className="flex justify-center items-center h-screen bg-dark-bg text-white text-xl font-mono">Loading Arena...</div>;

  return (
    <div className="flex h-[calc(100vh-61px)] w-full font-sans relative overflow-hidden"> 
      
      {/* --- LEFT PANE --- */}
      <div className="w-1/2 flex flex-col border-r border-dark-border bg-dark-bg">
        <div className="flex items-center px-4 bg-[#1e1e1e] border-b border-dark-border select-none">
          <button 
            className={`py-3 px-4 text-sm font-bold transition ${leftTab === 'description' ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`}
            onClick={() => setLeftTab('description')}
          >Description</button>
          <button 
            className={`py-3 px-4 text-sm font-bold transition ${leftTab === 'history' ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`}
            onClick={() => setLeftTab('history')}
          >Submissions</button>
        </div>

        <div className="flex-grow p-6 overflow-y-auto custom-scrollbar">
          {leftTab === 'history' ? (
            <div className="bg-[#1e1e1e] border border-dark-border rounded-lg overflow-hidden shadow-xl">
              <table className="w-full text-left">
                <thead className="bg-[#2a2a2a] border-b border-dark-border text-gray-400 text-xs uppercase">
                  <tr>
                    <th className="p-4">Time</th>
                    <th className="p-4">Verdict</th>
                    <th className="p-4">Lang</th>
                    <th className="p-4 text-right">Action</th>
                  </tr>
                </thead>
                <tbody className="text-sm">
                  {history.map(sub => (
                    <tr key={sub.submission_id} className="border-b border-dark-border hover:bg-[#2a2a2a] transition">
                      <td className="p-4 text-gray-300">
                        {new Date(sub.submitted_at).toLocaleString([], { dateStyle: 'short', timeStyle: 'short' })}
                      </td>
                      <td className={`p-4 font-bold ${['AC', 'Accepted'].includes(sub.status) ? 'text-green-400' : 'text-red-400'}`}>
                        {sub.status}
                      </td>
                      <td className="p-4 text-gray-400 uppercase font-mono">{sub.language}</td>
                      <td className="p-4 text-right">
                        <button 
                          onClick={() => handleViewSubmission(sub.submission_id)}
                          className="text-dark-accent hover:underline text-xs font-bold"
                        >
                          View Code
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <>
            {/* Title Section with Edit Button */}
              <div className="flex justify-between items-start mb-3">
                <h2 className="text-3xl font-bold text-white tracking-tight">{problem.title}</h2>
                
                {canEdit && (
                  <Button 
                    variant="secondary" 
                    size="sm" 
                    onClick={() => navigate(`/edit-problem/${problem.problem_id}`)}
                    className="flex items-center space-x-2 border-dark-border hover:border-gray-500 shadow-lg"
                  >
                    <span>⚙️ Edit Problem</span>
                  </Button>
                )}
              </div>
              
              {/* ❌ REMOVED THE DUPLICATE <h2> TAG THAT WAS HERE */}
              
              {/* Badges */}
              <div className="flex space-x-3 mb-6">
                <span className="bg-[#1e1e1e] text-gray-400 px-3 py-1 rounded text-xs border border-dark-border">
                  ⏱️ {problem.time_limit_ms || 2000}ms
                </span>
                <span className="bg-[#1e1e1e] text-gray-400 px-3 py-1 rounded text-xs border border-dark-border">
                  💾 {problem.memory_limit_kb / 1024 || 256}MB
                </span>
                <span className={`px-3 py-1 text-xs rounded font-bold border ${
                  problem.difficulty === 'Easy' ? 'border-green-800 text-green-400' : 
                  problem.difficulty === 'Medium' ? 'border-yellow-800 text-yellow-400' : 'border-red-800 text-red-400'
                }`}>{problem.difficulty}</span>
              </div>
              
              {/* 👇 UPDATED MARKDOWN + MATH RENDERER */}
              {/* Note: Removed 'whitespace-pre-wrap' so Markdown handles spacing naturally */}
              <div className="prose prose-invert max-w-none text-gray-300 mb-8 text-[15px] leading-relaxed">
                <ReactMarkdown
                  remarkPlugins={[remarkMath, remarkGfm]} // 👈 ADD IT HERE
                  rehypePlugins={[rehypeKatex]}
                >
                  {problem.description || problemData.description}
                </ReactMarkdown>
              </div>

              {/* Latest Verdict Status Bar */}
              <div className="mt-auto p-4 bg-[#1a1a1a] border border-dark-border rounded-lg flex items-center justify-between">
                <span className="text-xs font-bold text-gray-500 uppercase tracking-widest">Verdict</span>
                <span className={`font-mono text-lg ${getStatusColor()}`}>{submitStatus || "Ready"}</span>
              </div>
            </>
          )}
        </div>
      </div>

      {/* --- RIGHT PANE --- */}
      <div className="w-1/2 flex flex-col bg-dark-surface border-l border-dark-border relative">
        <div className="flex justify-between items-center p-2 bg-[#1e1e1e] border-b border-dark-border z-10">
          <select 
            value={language}
            onChange={(e) => {
              setLanguage(e.target.value);
              setCode(boilerplates[e.target.value]);
            }}
            className="bg-dark-bg text-gray-300 px-3 py-1.5 rounded border border-dark-border font-mono text-sm outline-none"
          >
            <option value="cpp">C++ 20</option>
            <option value="python">Python 3</option>
            <option value="java">Java 17</option>
          </select>
          <div className="flex space-x-2">
            <Button onClick={handleRunCode} variant="secondary" size="sm">Run</Button>
            <Button onClick={handleSubmit} variant="success" size="sm">Submit</Button>
          </div>
        </div>

        <div className="flex-grow relative pb-11">
          <Editor
            height="100%"
            language={language === 'cpp' ? 'cpp' : language}
            theme="vs-dark"
            value={code}
            onChange={setCode}
            options={{ 
              fontSize: 15, 
              minimap: { enabled: false }, 
              padding: { top: 20 },
              scrollBeyondLastLine: false 
            }}
          />
        </div>

        {/* FLOATING CONSOLE */}
        <div className={`absolute bottom-0 left-0 w-full flex flex-col bg-[#1e1e1e]/95 backdrop-blur-sm border-t border-dark-border transition-all shadow-2xl z-20 ${isConsoleOpen ? 'h-72' : 'h-11'}`}>
          <div className="flex items-center justify-between px-4 py-2.5 cursor-pointer hover:bg-white/5 transition-colors" onClick={() => setIsConsoleOpen(!isConsoleOpen)}>
            <div className="flex space-x-6">
              <button onClick={(e) => { e.stopPropagation(); setActiveTab('input'); setIsConsoleOpen(true); }}
                className={`text-xs font-black uppercase tracking-widest ${activeTab === 'input' && isConsoleOpen ? 'text-white' : 'text-gray-500'}`}>Input</button>
              <button onClick={(e) => { e.stopPropagation(); setActiveTab('output'); setIsConsoleOpen(true); }}
                className={`text-xs font-black uppercase tracking-widest ${activeTab === 'output' && isConsoleOpen ? 'text-white' : 'text-gray-500'}`}>Output</button>
            </div>
            <span className="text-[10px] font-bold text-gray-600 uppercase tracking-widest">{isConsoleOpen ? 'Collapse' : 'Expand Console'}</span>
          </div>

          {isConsoleOpen && (
            <div className="flex-grow p-4 bg-[#0d0d0d]/90">
              {activeTab === 'input' ? (
                <textarea 
                  className="w-full h-full bg-transparent text-gray-300 font-mono text-sm outline-none resize-none"
                  placeholder="Custom stdin..."
                  value={customInput}
                  onChange={(e) => setCustomInput(e.target.value)}
                />
              ) : (
                <div className="h-full font-mono text-sm text-gray-300 overflow-y-auto whitespace-pre-wrap">
                  {consoleOutput || "No output to display."}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* --- SUBMISSION DETAILS MODAL --- */}
      {isModalOpen && selectedSubmission && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 backdrop-blur-md p-4 sm:p-8">
          <div className="bg-[#1e1e1e] w-full max-w-5xl h-[85vh] rounded-xl border border-dark-border flex flex-col shadow-2xl animate-in fade-in zoom-in duration-200">
            
            {/* Modal Header */}
            <div className="flex justify-between items-center p-5 border-b border-dark-border bg-[#252525]">
              <div className="flex items-center space-x-4">
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center space-x-3">
                    <span>Submission Details</span>
                    <span className={`text-[10px] px-2 py-0.5 rounded font-black uppercase tracking-tighter ${
                      ['AC', 'Accepted'].includes(selectedSubmission.status) ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                    }`}>
                      {selectedSubmission.status}
                    </span>
                  </h3>
                  <div className="flex items-center space-x-3 mt-1">
                    <p className="text-[10px] text-gray-500 font-mono">ID: {selectedSubmission.submission_id}</p>
                    <p className="text-[10px] text-gray-500 font-mono uppercase">Language: {selectedSubmission.language}</p>
                  </div>
                </div>
              </div>
              <button 
                onClick={() => setIsModalOpen(false)}
                className="p-2 hover:bg-white/10 rounded-full transition-colors text-gray-400 hover:text-white"
                title="Close"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
              </button>
            </div>

            {/* Modal Body: Read-Only Monaco Editor */}
            <div className="flex-grow relative bg-[#1e1e1e]">
              <Editor
                height="100%"
                language={selectedSubmission.language === 'cpp' ? 'cpp' : selectedSubmission.language}
                theme="vs-dark"
                value={selectedSubmission.source_code}
                options={{ 
                  readOnly: true, 
                  fontSize: 14, 
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                  padding: { top: 20 }
                }}
              />
            </div>

            {/* Modal Footer */}
            <div className="p-4 border-t border-dark-border bg-[#252525] flex justify-between items-center">
              <div className="text-xs text-gray-500 italic">
                {selectedSubmission.message && `Logs: ${selectedSubmission.message.substring(0, 70)}...`}
              </div>
              <div className="flex space-x-3">
                <Button 
                  variant="secondary" 
                  onClick={() => setIsModalOpen(false)}
                >
                  Close
                </Button>
                <Button 
                  variant="success" 
                  className="flex items-center space-x-2"
                  onClick={() => {
                    setCode(selectedSubmission.source_code);
                    setLanguage(selectedSubmission.language);
                    setIsModalOpen(false);
                  }}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path><path d="M3 3v5h5"></path></svg>
                  <span>Restore to Editor</span>
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}