import React from 'react';
import remarkGfm from 'remark-gfm';
import ReactMarkdown from 'react-markdown';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import Button from '../../components/ui/Button';

export default function ProblemDescription({ problem, canEdit, navigate, submitStatus }) {
  const getStatusColor = () => {
    if (['Accepted', 'AC'].includes(submitStatus)) return 'text-green-400 font-bold';
    if (['WA', 'CE', 'RE', 'TLE', 'SE'].includes(submitStatus) || submitStatus.includes('Error')) return 'text-red-400 font-bold';
    if (['Pending', 'Running'].includes(submitStatus) || submitStatus.includes('⏳')) return 'text-yellow-400 animate-pulse';
    return 'text-gray-400';
  };

  return (
    <>
      <div className="flex justify-between items-start mb-3">
        <h2 className="text-3xl font-bold text-white tracking-tight">{problem.title}</h2>
        {canEdit && (
          <Button variant="secondary" size="sm" onClick={() => navigate(`/edit-problem/${problem.problem_id}`)} className="flex items-center space-x-2 border-dark-border hover:border-gray-500 shadow-lg">
            <span>⚙️ Edit Problem</span>
          </Button>
        )}
      </div>
      
      <div className="flex space-x-3 mb-6">
        <span className="bg-[#1e1e1e] text-gray-400 px-3 py-1 rounded text-xs border border-dark-border">⏱️ {problem.time_limit_ms || 2000}ms</span>
        <span className="bg-[#1e1e1e] text-gray-400 px-3 py-1 rounded text-xs border border-dark-border">💾 {problem.memory_limit_kb / 1024 || 256}MB</span>
        <span className={`px-3 py-1 text-xs rounded font-bold border ${problem.difficulty === 'Easy' ? 'border-green-800 text-green-400' : problem.difficulty === 'Medium' ? 'border-yellow-800 text-yellow-400' : 'border-red-800 text-red-400'}`}>{problem.difficulty}</span>
      </div>
      
      <div className="prose prose-invert max-w-none text-gray-300 mb-8 text-[15px] leading-relaxed">
        <ReactMarkdown remarkPlugins={[remarkMath, remarkGfm]} rehypePlugins={[rehypeKatex]}>
          {problem.description}
        </ReactMarkdown>
      </div>

      <div className="mt-auto p-4 bg-[#1a1a1a] border border-dark-border rounded-lg flex items-center justify-between">
        <span className="text-xs font-bold text-gray-500 uppercase tracking-widest">Verdict</span>
        <span className={`font-mono text-lg ${getStatusColor()}`}>{submitStatus || "Ready"}</span>
      </div>
    </>
  );
}