import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import ProblemDescription from './ProblemDescription';
import SubmissionHistory from './SubmissionHistory';
import CodeEditor from './CodeEditor';
import ExecutionConsole from './ExecutionConsole';
// 👇 1. Import some icons for the new banner
import { ArrowLeft, LogOut, ShieldAlert } from 'lucide-react'; 

const boilerplates = {
  cpp: `#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n    // Write your C++ code here\n    return 0;\n}`,
  python: `# Write your Python code here`,
  java: `import java.util.*;\nimport java.io.*;\n\n public class Main {\n    public static void main(String[] args) {\n        // Write your Java code here\n    }\n}`
};

export default function ContestArena() {
  const { id: contestId, problemId } = useParams(); 
  const navigate = useNavigate();
  const { currentUser } = useAuth();
  
  const [problem, setProblem] = useState(null);
  const [code, setCode] = useState(boilerplates['cpp']);
  const [language, setLanguage] = useState('cpp');
  const [submitStatus, setSubmitStatus] = useState(''); 
  const [leftTab, setLeftTab] = useState('description');
  const [history, setHistory] = useState([]);
  
  const [isConsoleOpen, setIsConsoleOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('input');
  const [customInput, setCustomInput] = useState('');
  const [consoleOutput, setConsoleOutput] = useState('');

  const draftKey = currentUser ? `draft_${currentUser.id}_${problemId}` : null;

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
    setCode(boilerplates['cpp']);
  }, [problemId, draftKey]);

  useEffect(() => {
    if (!draftKey || !code) return;
    const timer = setTimeout(() => {
      if (code !== boilerplates[language]) {
         localStorage.setItem(draftKey, JSON.stringify({ language, code }));
      }
    }, 1000); 
    return () => clearTimeout(timer);
  }, [code, language, draftKey]);

  useEffect(() => {
    api.get(`/problems/${problemId}`)
      .then(res => setProblem(res.data))
      .catch(err => console.error("Could not fetch problem details", err));
    fetchHistory();
  }, [problemId]);

  const fetchHistory = async () => {
    try {
      const res = await api.get(`/submissions/history/${problemId}?contest_id=${contestId}`);
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
        problem_id: problemId, 
        contest_id: contestId, 
        language, 
        source_code: code 
      });
      
      const token = localStorage.getItem('token');
      const sse = new EventSource(`${api.defaults.baseURL}/submissions/stream/${response.data.submission_id}?token=${token}`);
      
      sse.onmessage = (event) => {
        const data = JSON.parse(event.data);
        const status = data.status;
        
        if (status === 'Running') {
          setSubmitStatus('Running... ⚙️');
        } else {
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
            if (draftKey) localStorage.removeItem(draftKey);
          }
          sse.close(); 
        }
      };

      sse.onerror = () => { sse.close(); };
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
      const token = localStorage.getItem('token');
      const sse = new EventSource(`${api.defaults.baseURL}/run/stream/${response.data.run_id}?token=${token}`);
      
      sse.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.status === 'Running') {
          setConsoleOutput('Running... ⚙️');
        } else if (data.status === 'Completed' || data.status === 'CE' || data.status === 'RE' || data.status === 'TLE' || data.status === 'SE') {
          setConsoleOutput(data.output || data.message || "Program finished with no output.");
          sse.close(); 
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

  if (!problem) return <div className="flex justify-center items-center h-screen bg-dark-bg text-white text-xl font-mono">Loading Contest Arena...</div>;

  return (
    // 👇 2. Changed h-[calc(100vh-61px)] to h-screen
    <div className="flex flex-col h-screen w-full font-sans relative overflow-hidden bg-dark-bg">
      
      {/* 👇 3. UPGRADED TOP BANNER */}
      <div className="bg-[#1a1a1a] border-b border-dark-border px-6 py-3 flex justify-between items-center shadow-md z-20">
        
        <div className="flex items-center gap-6">
          {/* Security Indicator */}
          <div className="text-red-500 font-mono font-bold tracking-widest flex items-center gap-2 text-sm bg-red-950/30 px-3 py-1.5 rounded border border-red-900/50">
            <span className="w-2.5 h-2.5 rounded-full bg-red-500 animate-pulse"></span>
            <ShieldAlert size={16} />
            SECURE EXAM ENVIRONMENT
          </div>
          
          {/* Faculty Escape Hatch */}
          {(currentUser?.role === 'admin' || currentUser?.role === 'professor') && (
            <button 
              onClick={() => navigate('/contests')} 
              className="flex items-center gap-2 text-red-300 hover:text-white font-bold bg-red-900/40 hover:bg-red-700 px-3 py-1.5 rounded border border-red-700/50 transition-colors text-sm shadow-sm"
              title="Return to Faculty Dashboard"
            >
              <LogOut size={16} />
              Exit to Workspace
            </button>
          )}
        </div>

        {/* Big Return to Hub Button */}
        <button 
          onClick={() => navigate(`/contests/${contestId}/arena`)} 
          className="flex items-center gap-2 bg-[#2a2a2a] hover:bg-gray-700 text-gray-200 hover:text-white px-4 py-2 rounded font-bold border border-dark-border transition-all shadow-sm text-sm"
        >
          <ArrowLeft size={16} />
          Return to Hub
        </button>
      </div>

      {/* Editor & Content Area (flex-1 lets it take the remaining height perfectly) */}
      <div className="flex flex-1 w-full relative overflow-hidden"> 
        <div className="w-1/2 flex flex-col border-r border-dark-border bg-dark-bg">
          <div className="flex items-center px-4 bg-[#1e1e1e] border-b border-dark-border select-none">
            <button className={`py-3 px-4 text-sm font-bold transition ${leftTab === 'description' ? 'text-white border-b-2 border-blue-500' : 'text-gray-400 hover:text-white'}`} onClick={() => setLeftTab('description')}>Description</button>
            <button className={`py-3 px-4 text-sm font-bold transition ${leftTab === 'history' ? 'text-white border-b-2 border-blue-500' : 'text-gray-400 hover:text-white'}`} onClick={() => setLeftTab('history')}>Submissions</button>
          </div>
          <div className="flex-grow p-6 overflow-y-auto custom-scrollbar">
            {leftTab === 'history' ? (
              <SubmissionHistory history={history} setCode={setCode} setLanguage={setLanguage} />
            ) : (
              <ProblemDescription problem={problem} canEdit={false} navigate={navigate} submitStatus={submitStatus} />
            )}
          </div>
        </div>

        <div className="w-1/2 flex flex-col bg-dark-surface border-l border-dark-border relative">
          <CodeEditor 
            code={code} setCode={setCode} language={language} setLanguage={setLanguage} 
            boilerplates={boilerplates} onRun={handleRunCode} onSubmit={handleSubmit}
            isContest={true} contestId={contestId} 
          />
          <ExecutionConsole 
            isConsoleOpen={isConsoleOpen} setIsConsoleOpen={setIsConsoleOpen}
            activeTab={activeTab} setActiveTab={setActiveTab}
            customInput={customInput} setCustomInput={setCustomInput}
            consoleOutput={consoleOutput}
          />
        </div>
      </div>
    </div>
  );
}