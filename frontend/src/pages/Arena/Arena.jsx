import React, { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import ProblemDescription from './ProblemDescription';
import SubmissionHistory from './SubmissionHistory';
import CodeEditor from './CodeEditor';
import ExecutionConsole from './ExecutionConsole';

const boilerplates = {
  cpp: `#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n    // Write your C++ code here\n    return 0;\n}`,
  python: `# Write your Python code here`,
  java: `import java.util.*;\nimport java.io.*;\n\n public class Main {\n    public static void main(String[] args) {\n        // Write your Java code here\n    }\n}`
};

export default function Arena() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { currentUser } = useAuth();
  
  // State
  const [problem, setProblem] = useState(null);
  const [code, setCode] = useState(boilerplates['cpp']);
  const [language, setLanguage] = useState('cpp');
  const [submitStatus, setSubmitStatus] = useState(''); 
  const [leftTab, setLeftTab] = useState('description');
  const [history, setHistory] = useState([]);
  
  // Console State
  const [isConsoleOpen, setIsConsoleOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('input');
  const [customInput, setCustomInput] = useState('');
  const [consoleOutput, setConsoleOutput] = useState('');

  // --- DRAFT LOGIC ---
  const draftKey = currentUser ? `draft_${currentUser.id}_${id}` : null;

  // 1. Load Draft on Mount
  useEffect(() => {
    if (draftKey) {
      const savedDraft = localStorage.getItem(draftKey);
      if (savedDraft) {
        try {
          const parsed = JSON.parse(savedDraft);
          setLanguage(parsed.language);
          setCode(parsed.code);
          return; 
        } catch (e) {
          console.error("Failed to parse draft", e);
        }
      }
    }
    // Fallback to boilerplate if no draft exists
    setCode(boilerplates['cpp']);
  }, [id, draftKey]);

  // 2. Save Draft on Change (Debounced)
  useEffect(() => {
    if (!draftKey || !code) return;

    // Wait 1 second after the user stops typing before saving to localStorage
    const timer = setTimeout(() => {
      // Don't save if they haven't modified the boilerplate
      if (code !== boilerplates[language]) {
         localStorage.setItem(draftKey, JSON.stringify({ language, code }));
      }
    }, 1000); 

    return () => clearTimeout(timer);
  }, [code, language, draftKey]);

  useEffect(() => {
    api.get(`/problems/${id}`)
      .then(res => setProblem(res.data))
      .catch(err => console.error("Could not fetch problem details", err));
    fetchHistory();
  }, [id]);

  const canEdit = currentUser && problem && (currentUser.role === 'admin' || (currentUser.role === 'professor' && currentUser.id === problem.author_id));

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
      const response = await api.post('/submit', { problem_id: id, language, source_code: code });
      
      // 👇 NEW: Start SSE Stream
      const token = localStorage.getItem('token');
      const sse = new EventSource(`${api.defaults.baseURL}/submissions/stream/${response.data.submission_id}?token=${token}`);
      
      sse.onmessage = (event) => {
        const data = JSON.parse(event.data);
        const status = data.status;
        
        if (status === 'Running') {
          setSubmitStatus('Running... ⚙️');
        } else {
          // Terminal state reached
          setSubmitStatus(status);
          fetchHistory();
          
          if (['CE', 'RE', 'WA', 'TLE', 'SE'].includes(status)) {
            setConsoleOutput(data.message || `Verdict: ${status}`);
            setActiveTab('output');
            setIsConsoleOpen(true);
          } else if (status === 'AC' || status === 'Accepted') {
            setConsoleOutput("Execution Successful! 🎉\nAll test cases passed.");
            setActiveTab('output');
            setIsConsoleOpen(true);
            
            // 3. Clear Draft on Accepted Answer
            if (draftKey) {
              localStorage.removeItem(draftKey);
            }
          }
          
          sse.close(); // Close connection
        }
      };

      sse.onerror = () => {
        console.error("SSE connection lost.");
        sse.close();
      };

    } catch (error) {
      setSubmitStatus('Error: Submission Failed');
    }
  };

  const handleRunCode = async () => {
    setConsoleOutput('Queuing... ⚙️');
    setActiveTab('output');
    setIsConsoleOpen(true);
    try {
      const response = await api.post('/run', { language, source_code: code, custom_input: customInput });
      
      // 👇 NEW: Start SSE Stream for Custom Run
      const token = localStorage.getItem('token');
      const sse = new EventSource(`${api.defaults.baseURL}/run/stream/${response.data.run_id}?token=${token}`);
      
      sse.onmessage = (event) => {
        const data = JSON.parse(event.data);
        
        if (data.status === 'Running') {
          setConsoleOutput('Running... ⚙️');
        } else if (data.status === 'Completed' || data.status === 'CE' || data.status === 'RE' || data.status === 'TLE' || data.status === 'SE') {
          setConsoleOutput(data.output || data.message || "Program finished with no output.");
          sse.close(); // Close connection
        }
      };

      sse.onerror = () => {
        setConsoleOutput('Error streaming execution status.');
        sse.close();
      };

    } catch (error) {
      setConsoleOutput('Error: Could not connect to execution engine.');
    }
  };

  if (!problem) return <div className="flex justify-center items-center h-screen bg-dark-bg text-white text-xl font-mono">Loading Arena...</div>;

  return (
    <div className="flex h-[calc(100vh-61px)] w-full font-sans relative overflow-hidden"> 
      {/* LEFT PANE */}
      <div className="w-1/2 flex flex-col border-r border-dark-border bg-dark-bg">
        <div className="flex items-center px-4 bg-[#1e1e1e] border-b border-dark-border select-none">
          <button className={`py-3 px-4 text-sm font-bold transition ${leftTab === 'description' ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`} onClick={() => setLeftTab('description')}>Description</button>
          <button className={`py-3 px-4 text-sm font-bold transition ${leftTab === 'history' ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`} onClick={() => setLeftTab('history')}>Submissions</button>
        </div>
        <div className="flex-grow p-6 overflow-y-auto custom-scrollbar">
          {leftTab === 'history' ? (
            <SubmissionHistory history={history} setCode={setCode} setLanguage={setLanguage} />
          ) : (
            <ProblemDescription problem={problem} canEdit={canEdit} navigate={navigate} submitStatus={submitStatus} />
          )}
        </div>
      </div>

      {/* RIGHT PANE */}
      <div className="w-1/2 flex flex-col bg-dark-surface border-l border-dark-border relative">
        <CodeEditor 
          code={code} setCode={setCode} language={language} setLanguage={setLanguage} 
          boilerplates={boilerplates} onRun={handleRunCode} onSubmit={handleSubmit}
          isContest={isContest} contestId={id}
        />
        <ExecutionConsole 
          isConsoleOpen={isConsoleOpen} setIsConsoleOpen={setIsConsoleOpen}
          activeTab={activeTab} setActiveTab={setActiveTab}
          customInput={customInput} setCustomInput={setCustomInput}
          consoleOutput={consoleOutput}
        />
      </div>
    </div>
  );
}