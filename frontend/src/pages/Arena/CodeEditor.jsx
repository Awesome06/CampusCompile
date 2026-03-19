import React, { useRef, useEffect, useState } from 'react';
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
  const isInternalChange = useRef(false);
  const [showResetModal, setShowResetModal] = useState(false);
  
  // Anti-Cheat memory leak prevention refs
  const disposablesRef = useRef([]);
  const domListenersRef = useRef([]);

  // Generic cleanup on unmount
  useEffect(() => {
    return () => {
      disposablesRef.current.forEach(d => { if (d && d.dispose) d.dispose(); });
      domListenersRef.current.forEach(({ element, type, handler, capture }) => {
        if (element) element.removeEventListener(type, handler, capture);
      });
      disposablesRef.current = [];
      domListenersRef.current = [];
    };
  }, []);

  const handleEditorMount = (editor, monaco) => {
    editorRef.current = editor;

    if (isContest) {
      disposablesRef.current.push(editor.onKeyDown((e) => {
        logKeystroke(); 
      }));

      editor.updateOptions({ contextmenu: false });

      const preventPaste = (e) => {
        e.preventDefault();
        e.stopPropagation();
        logPasteAttempt(); 
      };

      const preventCopy = (e) => {
        e.preventDefault();
        e.stopPropagation();
      };

      // 1. Block native keyboard shortcuts (Ctrl+V / Cmd+V)
      editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyV, () => {
        logPasteAttempt();
      });
      // Optionally block copy/cut shortcuts
      editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyC, () => {});
      editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyX, () => {});

      const domNode = editor.getDomNode();
      const textarea = domNode.querySelector('textarea');

      // Helper function to attach and track DOM events
      const attachEvent = (element, type, handler) => {
        if (!element) return;
        element.addEventListener(type, handler, true);
        domListenersRef.current.push({ element, type, handler, capture: true });
      };

      // 2. Intercept native DOM events on the wrapper
      attachEvent(domNode, 'paste', preventPaste);
      attachEvent(domNode, 'drop', preventPaste);
      attachEvent(domNode, 'copy', preventCopy);
      attachEvent(domNode, 'cut', preventCopy);

      // 3. Intercept Monaco's hidden textarea (where edits actually occur)
      if (textarea) {
        attachEvent(textarea, 'paste', preventPaste);
        attachEvent(textarea, 'drop', preventPaste);
        attachEvent(textarea, 'copy', preventCopy);
        attachEvent(textarea, 'cut', preventCopy);
      }

      // 4. Final safety net: Monaco's internal onDidPaste event
      disposablesRef.current.push(editor.onDidPaste(() => {
        logPasteAttempt();
      }));
    }
  };

  const handleEditorChange = (value) => {
    isInternalChange.current = true; // Throw up the shield: we are typing!
    setCode(value);
  };

  const handleReset = () => {
    setShowResetModal(true);
  };

  // Only inject external code (like restoring history or resetting)
  // If the user is actively typing, block the parent from hijacking the editor.
  useEffect(() => {
    if (editorRef.current && !isInternalChange.current) {
      const currentEditorValue = editorRef.current.getValue();
      if (currentEditorValue !== code) {
        // Only reset the cursor if we are legitimately loading new code from the outside
        editorRef.current.setValue(code || boilerplates[language]); 
      }
    }
    // Reset the flag after every state evaluation
    isInternalChange.current = false;
  }, [code, language, boilerplates]);

  return (
    <>
      {showResetModal && (
        <div className="fixed inset-0 z-[100] bg-black/80 flex items-center justify-center backdrop-blur-sm">
          <div className="bg-dark-surface border border-dark-border p-6 rounded-lg max-w-sm w-full shadow-2xl">
            <h3 className="text-xl font-bold mb-4 text-white flex items-center gap-2">
              <RefreshCw className="text-red-500" size={24} /> Reset Editor?
            </h3>
            <p className="text-gray-300 mb-6 text-sm">
              Are you sure you want to reset the editor to the default template? Your current code will be permanently lost.
            </p>
            <div className="flex justify-end gap-3">
              <button 
                onClick={() => setShowResetModal(false)}
                className="px-4 py-2 rounded text-sm font-bold text-gray-300 hover:bg-gray-700 transition"
              >
                Cancel
              </button>
              <button 
                onClick={() => {
                  setCode(boilerplates[language]);
                  setShowResetModal(false);
                }}
                className="px-4 py-2 rounded text-sm font-bold bg-red-600 hover:bg-red-500 text-white transition"
              >
                Reset Code
              </button>
            </div>
          </div>
        </div>
      )}

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
          defaultValue={code}
          onChange={handleEditorChange}
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