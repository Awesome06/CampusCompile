import React, { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import api from '../../services/api';
import Button from '../../components/ui/Button';

export default function SubmissionHistory({ history, setCode, setLanguage, hasMoreHistory, historyOffset, historyLimit, fetchPrevHistory, fetchNextHistory }) {
  const [selectedSubmission, setSelectedSubmission] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const displayStart = history.length > 0 ? historyOffset + 1 : 0;
  const displayEnd = historyOffset + history.length;

  useEffect(() => {
    const handleKeyDown = (e) => {
      // Don't trigger if user is typing
      if (document.activeElement.tagName === 'INPUT' || document.activeElement.tagName === 'TEXTAREA') return;
      if (e.key === 'ArrowLeft') {
        if (fetchPrevHistory) fetchPrevHistory();
      } else if (e.key === 'ArrowRight') {
        if (fetchNextHistory) fetchNextHistory();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [historyOffset, hasMoreHistory, fetchPrevHistory, fetchNextHistory]);

  const handleViewSubmission = async (submissionId) => {
    try {
      const res = await api.get(`/submissions/${submissionId}`);
      setSelectedSubmission(res.data);
      setIsModalOpen(true);
    } catch (err) {
      console.error("Error fetching submission details:", err);
    }
  };

  return (
    <>
      <div className="bg-[#1e1e1e] border border-dark-border rounded-lg overflow-hidden shadow-xl">
        <table className="w-full text-left">
          <thead className="bg-[#2a2a2a] border-b border-dark-border text-gray-400 text-xs uppercase">
            <tr>
              <th className="p-4">Time</th>
              <th className="p-4">Verdict</th>
              <th className="p-4">Lang</th>
              <th className="p-4 text-right">Action</th>
            </tr>
          </thead>
          <tbody className="text-sm">
            {history.map(sub => (
              <tr key={sub.submission_id} className="border-b border-dark-border hover:bg-[#2a2a2a] transition">
                <td className="p-4 text-gray-300">{new Date(sub.submitted_at).toLocaleString([], { dateStyle: 'short', timeStyle: 'short' })}</td>
                <td className={`p-4 font-bold ${['AC', 'Accepted'].includes(sub.status) ? 'text-green-400' : 'text-red-400'}`}>{sub.status}</td>
                <td className="p-4 text-gray-400 uppercase font-mono">{sub.language}</td>
                <td className="p-4 text-right">
                  <button onClick={() => handleViewSubmission(sub.submission_id)} className="text-dark-accent hover:underline text-xs font-bold">View Code</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {(hasMoreHistory || historyOffset > 0) && (
          <div className="flex justify-between items-center py-3 px-4 border-t border-dark-border bg-[#1e1e1e]">
            <Button onClick={fetchPrevHistory} variant="outline" disabled={historyOffset === 0} className="w-1/5 text-gray-300 border-dark-border hover:bg-[#2a2a2a] disabled:opacity-30 disabled:cursor-not-allowed text-xs py-1 px-4">
              &larr; Prev
            </Button>
            <span className="w-3/5 text-center text-gray-400 text-xs font-mono">
              Showing {displayStart} - {displayEnd}
            </span>
            <Button onClick={fetchNextHistory} variant="outline" disabled={!hasMoreHistory} className="w-1/5 text-gray-300 border-dark-border hover:bg-[#2a2a2a] disabled:opacity-30 disabled:cursor-not-allowed text-xs py-1 px-4">
              Next &rarr;
            </Button>
          </div>
        )}
      </div>

      {isModalOpen && selectedSubmission && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 backdrop-blur-md p-4 sm:p-8">
          <div className="bg-[#1e1e1e] w-full max-w-5xl h-[85vh] rounded-xl border border-dark-border flex flex-col shadow-2xl animate-in fade-in zoom-in duration-200">
            <div className="flex justify-between items-center p-5 border-b border-dark-border bg-[#252525]">
              <div className="flex items-center space-x-4">
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center space-x-3">
                    <span>Submission Details</span>
                    <span className={`text-[10px] px-2 py-0.5 rounded font-black uppercase tracking-tighter ${['AC', 'Accepted'].includes(selectedSubmission.status) ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>{selectedSubmission.status}</span>
                  </h3>
                  <div className="flex items-center space-x-3 mt-1">
                    <p className="text-[10px] text-gray-500 font-mono">ID: {selectedSubmission.submission_id}</p>
                    <p className="text-[10px] text-gray-500 font-mono uppercase">Language: {selectedSubmission.language}</p>
                  </div>
                </div>
              </div>
              <button onClick={() => setIsModalOpen(false)} className="p-2 hover:bg-white/10 rounded-full transition-colors text-gray-400 hover:text-white" title="Close">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
              </button>
            </div>
            <div className="flex-grow relative bg-[#1e1e1e]">
              <Editor
                height="100%" language={selectedSubmission.language === 'cpp' ? 'cpp' : selectedSubmission.language}
                theme="vs-dark" value={selectedSubmission.source_code}
                options={{ readOnly: true, fontSize: 14, minimap: { enabled: false }, scrollBeyondLastLine: false, automaticLayout: true, padding: { top: 20 } }}
              />
            </div>
            <div className="p-4 border-t border-dark-border bg-[#252525] flex justify-between items-center">
              <div className="text-xs text-gray-500 italic">{selectedSubmission.message && `Logs: ${selectedSubmission.message.substring(0, 70)}...`}</div>
              <div className="flex space-x-3">
                <Button variant="secondary" onClick={() => setIsModalOpen(false)}>Close</Button>
                <Button variant="success" className="flex items-center space-x-2" onClick={() => { setCode(selectedSubmission.source_code); setLanguage(selectedSubmission.language); setIsModalOpen(false); }}>
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path><path d="M3 3v5h5"></path></svg>
                  <span>Restore to Editor</span>
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}