import React from 'react';

export default function ExecutionConsole({ 
  isConsoleOpen, setIsConsoleOpen, 
  activeTab, setActiveTab, 
  customInput, setCustomInput, 
  consoleOutput 
}) {
  return (
    <div className={`absolute bottom-0 left-0 w-full flex flex-col bg-[#1e1e1e]/95 backdrop-blur-sm border-t border-dark-border transition-all shadow-2xl z-20 ${isConsoleOpen ? 'h-72' : 'h-11'}`}>
      <div className="flex items-center justify-between px-4 py-2.5 cursor-pointer hover:bg-white/5 transition-colors" onClick={() => setIsConsoleOpen(!isConsoleOpen)}>
        <div className="flex space-x-6">
          <button 
            onClick={(e) => { e.stopPropagation(); setActiveTab('input'); setIsConsoleOpen(true); }}
            className={`text-xs font-black uppercase tracking-widest ${activeTab === 'input' && isConsoleOpen ? 'text-white' : 'text-gray-500'}`}
          >
            Input
          </button>
          <button 
            onClick={(e) => { e.stopPropagation(); setActiveTab('output'); setIsConsoleOpen(true); }}
            className={`text-xs font-black uppercase tracking-widest ${activeTab === 'output' && isConsoleOpen ? 'text-white' : 'text-gray-500'}`}
          >
            Output
          </button>
        </div>
        <span className="text-[10px] font-bold text-gray-600 uppercase tracking-widest">
          {isConsoleOpen ? 'Collapse' : 'Expand Console'}
        </span>
      </div>

      {isConsoleOpen && (
        <div className="flex-grow p-4 bg-[#0d0d0d]/90">
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
  );
}