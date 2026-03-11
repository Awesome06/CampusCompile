import React, { useRef } from 'react';
import Editor from '@monaco-editor/react';
import Button from '../../components/ui/Button';
import useAntiCheat from '../../hooks/useAntiCheat'; 
import { RefreshCw, Loader2 } from 'lucide-react'; // Added Loader2 icon

export default function CodeEditor({ 
  code, setCode, language, setLanguage, boilerplates, 
  onRun, onSubmit, isContest, contestId, isProcessing // 👈 Added isProcessing
}) {
  
  const { logPasteAttempt, logKeystroke } = useAntiCheat(contestId, isContest);
  const editorRef = useRef(null);

  const handleEditorMount = (editor, monaco) => {
    editorRef.current = editor;

    if (isContest) {
      editor.onKeyDown((e) => {
        logKeystroke(); 
      });

      editor.updateOptions({ contextmenu: false });

      const domNode = editor.getDomNode();

      const preventPaste = (e) => {
        e.preventDefault();
        e.stopPropagation();
        logPasteAttempt(); 
      };

      const preventCopy = (e) => {
        e.preventDefault();
        e.stopPropagation();
      };

      domNode.addEventListener('paste', preventPaste, true);
      domNode.addEventListener('drop', preventPaste, true);
      domNode.addEventListener('copy', preventCopy, true);
      domNode.addEventListener('cut', preventCopy, true);
    }
  };

  const handleReset = () => {
    if (window.confirm("Are you sure you want to reset the editor? Your current code will be permanently lost.")) {
      setCode(boilerplates[language]);
    }
  };

  return (
    <>
      <div className="flex justify-between items-center p-2 bg-[#1e1e1e] border-b border-dark-border z-10">
        <div className="flex items-center gap-3">
          <select 
            value={language}
            onChange={(e) => {
              setLanguage(e.target.value);
              setCode(boilerplates[e.target.value]);
            }}
            className="bg-dark-bg text-gray-300 px-3 py-1.5 rounded border border-dark-border font-mono text-sm outline-none"
            disabled={isProcessing} // Disable language switch while running
          >
            <option value="cpp">C++ 20</option>
            <option value="python">Python 3</option>
            <option value="java">Java 17</option>
          </select>
          
          <button
            onClick={handleReset}
            disabled={isProcessing}
            className={`p-1.5 rounded transition-colors shadow-sm ${isProcessing ? 'text-gray-600 bg-[#1a1a1a] cursor-not-allowed' : 'text-gray-400 hover:text-white bg-[#2a2a2a] hover:bg-[#3a3a3a] border border-dark-border'}`}
            title="Reset to boilerplate"
          >
            <RefreshCw size={16} />
          </button>

          {isContest && (
            <span className="text-[10px] font-bold text-red-500 bg-red-900/20 px-2 py-1 rounded border border-red-800 animate-pulse ml-2">
              ● CONTEST MODE SECURED
            </span>
          )}
        </div>

        {/* 👇 Button Lockouts */}
        <div className="flex space-x-2">
          <Button 
            onClick={onRun} 
            variant="secondary" 
            size="sm" 
            disabled={isProcessing}
            className={isProcessing ? 'opacity-50 cursor-not-allowed flex gap-2 items-center' : 'flex gap-2 items-center'}
          >
            {isProcessing && <Loader2 size={14} className="animate-spin" />}
            Run
          </Button>
          <Button 
            onClick={onSubmit} 
            variant="success" 
            size="sm" 
            disabled={isProcessing}
            className={isProcessing ? 'opacity-50 cursor-not-allowed flex gap-2 items-center' : 'flex gap-2 items-center'}
          >
            {isProcessing && <Loader2 size={14} className="animate-spin" />}
            Submit
          </Button>
        </div>
      </div>

      <div className="flex-grow relative pb-11">
        <Editor
          height="100%"
          language={language === 'cpp' ? 'cpp' : language}
          theme="vs-dark"
          value={code}
          onChange={setCode}
          onMount={handleEditorMount} 
          options={{ 
            fontSize: 15, 
            minimap: { enabled: false }, 
            padding: { top: 20 }, 
            scrollBeyondLastLine: false,
            wordWrap: "on",
            readOnly: isProcessing // Disable typing while processing
          }}
        />
      </div>
    </>
  );
}