import React, { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import ProblemDescription from './ProblemDescription';
import SubmissionHistory from './SubmissionHistory';
import CodeEditor from './CodeEditor';
import ExecutionConsole from './ExecutionConsole';
import useExecutionEngine, { boilerplates } from '../../hooks/useExecutionEngine';
import { ArrowLeft, LogOut, ShieldAlert } from 'lucide-react'; 

export default function ContestArena() {
  const { id: contestId, problemId } = useParams(); 
  const navigate = useNavigate();
  const { currentUser } = useAuth();
  const [leftTab, setLeftTab] = useState('description');
  
  // 👇 Connect the Brain and pass the contestId
  const {
    problem, code, setCode, language, setLanguage, submitStatus, history,
    isConsoleOpen, setIsConsoleOpen, activeTab, setActiveTab,
    customInput, setCustomInput, consoleOutput, isProcessing,
    handleSubmit, handleRunCode
  } = useExecutionEngine(problemId, contestId);

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
              <ProblemDescription problem={problem} canEdit={false} navigate={navigate} submitStatus={submitStatus} selectedLanguage={language}/>
            )}
          </div>
        </div>

        <div className="w-1/2 flex flex-col bg-dark-surface border-l border-dark-border relative">
          <CodeEditor 
            code={code} setCode={setCode} language={language} setLanguage={setLanguage} 
            boilerplates={boilerplates} onRun={handleRunCode} onSubmit={handleSubmit}
            isContest={true} contestId={contestId} isProcessing={isProcessing} // 👈 Passed down
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