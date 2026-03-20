import React, { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import ProblemDescription from './ProblemDescription';
import SubmissionHistory from './SubmissionHistory';
import CodeEditor from './CodeEditor';
import ExecutionConsole from './ExecutionConsole';
import useExecutionEngine, { boilerplates } from '../../hooks/useExecutionEngine';

export default function PracticeArena() {
  const { id } = useParams(); // id is the problem_id
  const navigate = useNavigate();
  const { currentUser } = useAuth();
  const [leftTab, setLeftTab] = useState('description');
  
  // 👇 Connect the Brain
  const {
    problem, code, setCode, language, setLanguage, submitStatus, history,
    hasMoreHistory, historyOffset, historyLimit, fetchPrevHistory, fetchNextHistory,
    isConsoleOpen, setIsConsoleOpen, activeTab, setActiveTab,
    customInput, setCustomInput, consoleOutput, isProcessing,
    handleSubmit, handleRunCode
  } = useExecutionEngine(id);

  const canEdit = currentUser && problem && (currentUser.role === 'admin' || (currentUser.role === 'professor' && currentUser.id === problem.author_id));

  if (!problem) return <div className="flex justify-center items-center h-[calc(100vh-61px)] bg-dark-bg text-white text-xl font-mono">Loading Practice Arena...</div>;

  return (
    <div className="flex h-[calc(100vh-61px)] w-full font-sans relative overflow-hidden"> 
      <div className="w-1/2 flex flex-col border-r border-dark-border bg-dark-bg">
        <div className="flex items-center px-4 bg-[#1e1e1e] border-b border-dark-border select-none">
          <button className={`py-3 px-4 text-sm font-bold transition ${leftTab === 'description' ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`} onClick={() => setLeftTab('description')}>Description</button>
          <button className={`py-3 px-4 text-sm font-bold transition ${leftTab === 'history' ? 'text-white border-b-2 border-dark-accent' : 'text-gray-400 hover:text-white'}`} onClick={() => setLeftTab('history')}>Submissions</button>
        </div>
        <div className="flex-grow p-6 overflow-y-auto custom-scrollbar">
          {leftTab === 'history' ? (
            <SubmissionHistory history={history} setCode={setCode} setLanguage={setLanguage} hasMoreHistory={hasMoreHistory} historyOffset={historyOffset} historyLimit={historyLimit} fetchPrevHistory={fetchPrevHistory} fetchNextHistory={fetchNextHistory} />
          ) : (
            <ProblemDescription problem={problem} canEdit={canEdit} navigate={navigate} submitStatus={submitStatus} selectedLanguage={language}/>
          )}
        </div>
      </div>

      <div className="w-1/2 flex flex-col bg-dark-surface border-l border-dark-border relative">
        <CodeEditor 
          code={code} setCode={setCode} language={language} setLanguage={setLanguage} 
          boilerplates={boilerplates} onRun={handleRunCode} onSubmit={handleSubmit}
          isContest={false} contestId={null} isProcessing={isProcessing} // 👈 Passed down here
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