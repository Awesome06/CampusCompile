import React, { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import { useParams } from 'react-router-dom';
import Latex from 'react-latex-next';
import api from '../services/api';
import Button from '../components/ui/Button';

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
  const [activeTab, setActiveTab] = useState('input');
  const [customInput, setCustomInput] = useState('');
  const [consoleOutput, setConsoleOutput] = useState('');
  const [leftTab, setLeftTab] = useState('description');
  const [history, setHistory] = useState([]);
  const [selectedSubmission, setSelectedSubmission] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  useEffect(() => {
    api.get(`/problems/${id}`)
      .then(res => setProblem(res.data))
      .catch(err => console.error("Could not fetch problem details", err));
      
    fetchHistory();
  }, [id]);

  const fetchHistory = async () => {
    try {
      const res = await api.get(`/submissions/history/${id}`);
      setHistory(res.data || []);
    } catch (err) {
      console.error("Could not fetch history:", err);
    }
  };

  const formatText = (text) => text ? text.replace(/\\n/g, '\n') : "";

  const handleCopy = (text) => navigator.clipboard.writeText(formatText(text));

  const handleSubmit = async () => {
    setSubmitStatus('Pending... ⏳');
    setIsConsoleOpen(false);
    
    try {
      // Matches Go SubmitRequest struct
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

        // Handle error displays for CE, RE, WA, TLE
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
      // Matches Go RunRequest struct
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
        // If there was an error (CE, RE), show the message, otherwise show output
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
    <div className="flex h-[calc(100vh-61px)] w-full font-sans"> 
      {/* LEFT PANE: Description & History */}
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
                    <th className="p-4 text-right">Action</th> {/* 👈 NEW COLUMN */}
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
              <h2 className="text-3xl font-bold mb-3 text-white tracking-tight">{problem.title}</h2>
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
              
              <div className="prose prose-invert max-w-none text-gray-300 mb-8 whitespace-pre-wrap text-[15px] leading-relaxed">
                <Latex>{problem.description}</Latex>
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

      {/* RIGHT PANE: Monaco Editor & Console */}
      <div className="w-1/2 flex flex-col bg-dark-surface border-l border-dark-border">
        <div className="flex justify-between items-center p-2 bg-[#1e1e1e] border-b border-dark-border">
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

        <div className="flex-grow relative">
          <Editor
            height="100%"
            language={language === 'cpp' ? 'cpp' : language}
            theme="vs-dark"
            value={code}
            onChange={setCode}
            options={{ fontSize: 15, minimap: { enabled: false }, padding: { top: 20 } }}
          />
        </div>

        {/* CONSOLE AREA */}
        <div className={`flex flex-col bg-[#1e1e1e] border-t border-dark-border transition-all ${isConsoleOpen ? 'h-72' : 'h-11'}`}>
          <div className="flex items-center justify-between px-4 py-2.5 cursor-pointer" onClick={() => setIsConsoleOpen(!isConsoleOpen)}>
            <div className="flex space-x-6">
              <button onClick={(e) => { e.stopPropagation(); setActiveTab('input'); setIsConsoleOpen(true); }}
                className={`text-xs font-black uppercase tracking-widest ${activeTab === 'input' && isConsoleOpen ? 'text-white' : 'text-gray-500'}`}>Input</button>
              <button onClick={(e) => { e.stopPropagation(); setActiveTab('output'); setIsConsoleOpen(true); }}
                className={`text-xs font-black uppercase tracking-widest ${activeTab === 'output' && isConsoleOpen ? 'text-white' : 'text-gray-500'}`}>Output</button>
            </div>
            <span className="text-[10px] font-bold text-gray-600 uppercase">{isConsoleOpen ? 'Collapse' : 'Expand Console'}</span>
          </div>

          {isConsoleOpen && (
            <div className="flex-grow p-4 bg-[#0d0d0d]">
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
    </div>
  );
}