import React, { useRef } from 'react';
import Editor from '@monaco-editor/react';
import Button from '../../components/ui/Button';
import useAntiCheat from '../../hooks/useAntiCheat'; // <-- Import the new hook

export default function CodeEditor({ 
  code, setCode, language, setLanguage, boilerplates, 
  onRun, onSubmit, isContest, contestId 
}) {
  
  // Initialize the traps
  const { logPasteAttempt, logKeystroke } = useAntiCheat(contestId, isContest);
  const editorRef = useRef(null);

  // This fires once the Monaco Editor is fully loaded
  const handleEditorMount = (editor, monaco) => {
    editorRef.current = editor;

    // Intercept deep keypresses directly inside Monaco
    editor.onKeyDown((e) => {
      logKeystroke(); // Log every keypress for the AutoTyper math

      // Check for Paste combinations: Ctrl+V (Windows/Linux) or Cmd+V (Mac)
      if ((e.ctrlKey || e.metaKey) && e.keyCode === monaco.KeyCode.KeyV) {
        if (isContest) {
          e.preventDefault();   // Stop the paste
          e.stopPropagation();  // Stop event bubbling
          logPasteAttempt();    // Fire telemetry to the Go backend
          
          // Optional: You can show a local toast/alert here to the user
          // alert("Pasting is strictly disabled during active contests.");
        }
      }
    });

    // Disable right-click context menu entirely during contests
    if (isContest) {
      editor.updateOptions({ contextmenu: false });
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
          >
            <option value="cpp">C++ 20</option>
            <option value="python">Python 3</option>
            <option value="java">Java 17</option>
          </select>
          
          {/* Visual indicator so students know they are being monitored */}
          {isContest && (
            <span className="text-[10px] font-bold text-red-500 bg-red-900/20 px-2 py-1 rounded border border-red-800 animate-pulse">
              ● CONTEST MODE SECURED
            </span>
          )}
        </div>

        <div className="flex space-x-2">
          <Button onClick={onRun} variant="secondary" size="sm">Run</Button>
          <Button onClick={onSubmit} variant="success" size="sm">Submit</Button>
        </div>
      </div>

      <div className="flex-grow relative pb-11">
        <Editor
          height="100%"
          language={language === 'cpp' ? 'cpp' : language}
          theme="vs-dark"
          value={code}
          onChange={setCode}
          onMount={handleEditorMount} // <-- Wire up the interceptors
          options={{ 
            fontSize: 15, 
            minimap: { enabled: false }, 
            padding: { top: 20 }, 
            scrollBeyondLastLine: false,
            wordWrap: "on"
          }}
        />
      </div>
    </>
  );
}