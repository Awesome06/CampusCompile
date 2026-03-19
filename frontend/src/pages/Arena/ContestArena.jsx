import React, { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import ProblemDescription from './ProblemDescription';
import SubmissionHistory from './SubmissionHistory';
import CodeEditor from './CodeEditor';
import ExecutionConsole from './ExecutionConsole';
import useExecutionEngine, { boilerplates } from '../../hooks/useExecutionEngine';
import { ArrowLeft, LogOut, ShieldAlert, AlertTriangle, Maximize } from 'lucide-react';
import useSecureArena from '../../hooks/useSecureArena';

export default function ContestArena() {
  const { id: contestId, problemId } = useParams(); 
  const navigate = useNavigate();
  const [leftTab, setLeftTab] = useState('description');
  
  const [hasEnteredArena, setHasEnteredArena] = useState(false);
  const { isElevated, isFullscreen, tabViolations, toggleFullscreen } = useSecureArena(contestId, hasEnteredArena);

  const leaveTimeRef = useRef(null);
  const [timeAway, setTimeAway] = useState(0);

  useEffect(() => {
    // If student, and not elevated, and is fullscreen, then arena entered
    if (!isElevated && isFullscreen && !hasEnteredArena) {
      setHasEnteredArena(true);
    }
  }, [isElevated, isFullscreen, hasEnteredArena]);

  useEffect(() => {
    let interval;
    if (!isElevated && hasEnteredArena) {
        if (!isFullscreen) {
            if (!leaveTimeRef.current) leaveTimeRef.current = Date.now();
            interval = setInterval(() => {
                setTimeAway(Math.floor((Date.now() - leaveTimeRef.current) / 1000));
            }, 1000);
        } else {
            leaveTimeRef.current = null;
            setTimeAway(0);
        }
    }
    return () => clearInterval(interval);
  }, [isElevated, hasEnteredArena, isFullscreen]);

  const handleAcknowledgeLockout = async () => {
    await toggleFullscreen();
    navigate(`/contests/${contestId}/arena`);
  };

  const {
    problem, code, setCode, language, setLanguage, submitStatus, history,
    isConsoleOpen, setIsConsoleOpen, activeTab, setActiveTab,
    customInput, setCustomInput, consoleOutput, isProcessing,
    handleSubmit, handleRunCode
  } = useExecutionEngine(problemId, contestId);

  if (!problem) return <div className="flex justify-center items-center h-screen bg-dark-bg text-white text-xl font-mono">Loading Contest Arena...</div>;

  return (
    <div className={`font-sans relative bg-dark-bg text-white ${isFullscreen || (!isElevated && hasEnteredArena && !isFullscreen) ? 'h-screen w-screen overflow-hidden flex flex-col' : 'flex flex-col h-full w-full overflow-hidden'}`}>
      
      {!isElevated && hasEnteredArena && !isFullscreen && (
        <div className="fixed inset-0 h-screen w-screen bg-red-950 text-white flex flex-col items-center justify-center p-8 z-[9999]">
          <AlertTriangle size={80} className="text-yellow-500 mb-6 animate-pulse" />
          <h1 className="text-4xl font-bold mb-4 text-center">Environment Lockout</h1>
          <p className="text-xl mb-4 text-center text-red-200 max-w-2xl">
            You have exited the secure Arena. To maintain academic integrity, you must remain in Fullscreen mode. Continuing to leave the environment will flag your submission.
          </p>
          <div className="bg-red-900 border border-red-500 rounded-lg p-4 mb-8 text-center shadow-xl">
            <span className="block text-red-300 font-mono text-sm tracking-widest uppercase mb-1">Time Outside Arena</span>
            <span className="text-5xl font-mono font-bold text-white tracking-widest">{timeAway}s</span>
            <span className="block text-red-400 text-xs mt-2 italic">This incident is being recorded</span>
          </div>
          <button onClick={handleAcknowledgeLockout} className="bg-white text-red-900 px-8 py-4 rounded font-bold text-xl hover:bg-gray-200 transition shadow-lg">
            Acknowledge & Return to Hub
          </button>
        </div>
      )}

      {!isElevated && tabViolations > 0 && (
        <div className="bg-red-600 text-white text-center py-1 text-sm font-bold flex justify-center items-center gap-2 flex-shrink-0 z-50">
          <AlertTriangle size={14} /> Warning: Focus lost {tabViolations} time(s). This activity is being recorded.
        </div>
      )}

      {!isElevated && !hasEnteredArena && (
        <div className="absolute inset-0 z-[9999] bg-black/70 backdrop-blur-sm flex flex-col items-center justify-center">
          <div className="bg-dark-surface p-8 rounded-xl border border-blue-900 shadow-2xl text-center max-w-lg">
            <h2 className="text-3xl font-bold text-white mb-4">Ready to Begin?</h2>
            <p className="text-gray-300 mb-8">You must enter secure Arena Mode to view problems and submit code. Once started, exiting fullscreen will trigger a security lockout.</p>
            <button onClick={toggleFullscreen} className="bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded font-bold text-xl transition-colors w-full flex items-center justify-center gap-3 shadow-lg hover:scale-105 transform duration-200">
              <Maximize size={24} /> Enter Secure Arena
            </button>
          </div>
        </div>
      )}

      <div className="bg-[#1a1a1a] border-b border-dark-border px-6 py-3 flex justify-between items-center shadow-md z-20">
        <div className="flex items-center gap-6">
          <div className="text-red-500 font-mono font-bold tracking-widest flex items-center gap-2 text-sm bg-red-950/30 px-3 py-1.5 rounded border border-red-900/50">
            <span className="w-2.5 h-2.5 rounded-full bg-red-500 animate-pulse"></span>
            <ShieldAlert size={16} /> SECURE EXAM ENVIRONMENT
          </div>
          {isElevated && (
            <div className="flex items-center gap-3">
              <button 
                onClick={() => navigate('/contests')} 
                className="flex items-center gap-2 text-red-300 hover:text-white font-bold bg-red-900/40 hover:bg-red-700 px-3 py-1.5 rounded border border-red-700/50 transition-colors text-sm shadow-sm"
                title="Return to Faculty Dashboard"
              >
                <LogOut size={16} /> Exit to Workspace
              </button>
            </div>
          )}
        </div>
        <button onClick={() => navigate(`/contests/${contestId}/arena`)} className="flex items-center gap-2 bg-[#2a2a2a] hover:bg-gray-700 text-gray-200 hover:text-white px-4 py-2 rounded font-bold border border-dark-border transition-all shadow-sm text-sm">
          <ArrowLeft size={16} /> Return to Hub
        </button>
      </div>

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
            isContest={!isElevated} contestId={contestId} isProcessing={isProcessing}
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