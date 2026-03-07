import React from 'react';
import Editor from '@monaco-editor/react';
import Button from '../../components/ui/Button';

export default function CodeEditor({ code, setCode, language, setLanguage, boilerplates, onRun, onSubmit }) {
  return (
    <>
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
          options={{ fontSize: 15, minimap: { enabled: false }, padding: { top: 20 }, scrollBeyondLastLine: false }}
        />
      </div>
    </>
  );
}