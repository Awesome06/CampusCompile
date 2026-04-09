import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import ReactMarkdown from 'react-markdown';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import remarkGfm from 'remark-gfm';
import api from '../services/api'; 
import Button from '../components/ui/Button';
import { parseZipTestCases } from '../utils/testCaseParser';
import { markdownComponents } from '../utils/markdownConfig';

export default function EditProblem() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [currentUser, setCurrentUser] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [step, setStep] = useState(1);
  const [status, setStatus] = useState({ type: '', message: '' });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const [problemData, setProblemData] = useState({
    title: '', description: '', difficulty: 'Easy', 
    time_limit: 2000, memory_limit: 256, is_public: false
  });

  const [testCases, setTestCases] = useState([]);
  const [expandedCases, setExpandedCases] = useState({ 0: true });

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) {
      navigate('/login');
      return;
    }

    try {
      const decoded = jwtDecode(token);
      const user = { id: decoded.user_id || decoded.sub, role: decoded.role?.toLowerCase() };
      setCurrentUser(user);

      const loadProblemData = async () => {
        try {
          const probRes = await api.get(`/problems/${id}`);
          const p = probRes.data;
          
          if (user.role !== 'admin' && user.id !== p.author_id) {
            navigate('/');
            return;
          }

          setProblemData({
            title: p.title, description: p.description, difficulty: p.difficulty,
            time_limit: p.time_limit_ms, memory_limit: p.memory_limit_kb / 1024, 
            is_public: p.is_public ?? false
          });

          const tcRes = await api.get(`/problems/${id}/testcases/all`);
          
          // 🛡️ THE FIX: Normalize Database snake_case to React camelCase immediately
          const normalizedTestCases = (tcRes.data.test_cases || []).map(tc => ({
            input: tc.input_data || '',
            expectedOutput: tc.expected_output || '',
            isHidden: tc.is_hidden ?? true,
            isSample: tc.is_sample ?? false
          }));
          
          setTestCases(normalizedTestCases);
          setIsLoading(false);
        } catch (err) {
          setStatus({ type: 'error', message: 'Could not load problem data.' });
          setIsLoading(false);
        }
      };

      loadProblemData();
    } catch (error) {
      navigate('/login');
    }
  }, [id, navigate]);

  // ACCORDION HANDLERS
  const toggleTestCase = (index) => {
    setExpandedCases(prev => ({ ...prev, [index]: !prev[index] }));
  };

  const handleAddTestCase = () => {
    const newIndex = testCases.length;
    setTestCases([...testCases, { input: '', expectedOutput: '', isHidden: true, isSample: false }]);
    setExpandedCases(prev => ({ ...prev, [newIndex]: true }));
  };

  const handleRemoveTestCase = (indexToRemove) => {
    // 1. Remove the test case from the main array
    setTestCases(prev => prev.filter((_, i) => i !== indexToRemove));

    // 2. Remap the expanded state to account for the shifted indices
    setExpandedCases(prev => {
      const newExpanded = {};
      Object.keys(prev).forEach(key => {
        const numKey = parseInt(key, 10);
        
        if (numKey < indexToRemove) {
          // Items before the deleted index stay exactly where they are
          newExpanded[numKey] = prev[numKey];
        } else if (numKey > indexToRemove) {
          // Items after the deleted index shift left by 1
          newExpanded[numKey - 1] = prev[numKey];
        }
        // If numKey === indexToRemove, it gets dropped naturally
      });
      return newExpanded;
    });
  };

  const updateTestCase = (index, field, value) => {
    const updated = [...testCases];
    updated[index] = { ...updated[index], [field]: value };
    setTestCases(updated);
  };

  // 🛡️ ZIP PARSER FIX: Ignore macOS Ghost Files
  const handleZipUpload = async (e) => {
    e.preventDefault(); 
    const file = e.dataTransfer ? e.dataTransfer.files[0] : e.target.files[0];
    if (!file) return;
    
    setStatus({ type: 'info', message: 'Extracting test cases from ZIP...' });
    
    const { testCases: parsedCases, error } = await parseZipTestCases(file);
    
    if (error) {
      setStatus({ type: 'error', message: error });
    } else {
      const startingIndex = testCases.length;
      setTestCases(prev => [...prev, ...parsedCases]);
      
      // Expand the first newly added test case
      setExpandedCases(prev => ({ ...prev, [startingIndex]: true }));
      setStatus({ type: 'success', message: `Successfully appended ${parsedCases.length} test cases!` });
      setTimeout(() => setStatus({ type: '', message: '' }), 3000);
    }
    
    if(e.target) e.target.value = null; 
  };

const handleSaveChanges = async () => {
    setIsSubmitting(true);
    setStatus({ type: 'info', message: 'Saving problem and streaming test cases to S3... 🚀' });

    // 1. Build a SINGLE FormData object for the Atomic Batch
    const formData = new FormData();
    testCases.forEach((tc, index) => {
      const inBlob = tc.inputBlob || new Blob([tc.input], { type: 'text/plain' });
      const outBlob = tc.expectedBlob || new Blob([tc.expectedOutput], { type: 'text/plain' });
      
      // Appending to the same key creates an array on the backend!
      formData.append('input_files', inBlob, `in_${index}.txt`);
      formData.append('expected_files', outBlob, `out_${index}.txt`);
      formData.append('is_hidden', tc.isHidden ? 'true' : 'false');
      formData.append('is_sample', tc.isSample ? 'true' : 'false');
    });

    try {
      // 2. Update Problem Metadata (Sending both key formats to guarantee Go struct binds them)
      await api.put(`/problems/${id}`, {
        title: problemData.title,
        description: problemData.description,
        difficulty: problemData.difficulty,
        time_limit: problemData.time_limit,
        time_limit_ms: problemData.time_limit,
        memory_limit: problemData.memory_limit * 1024,
        memory_limit_kb: problemData.memory_limit * 1024,
        is_public: problemData.is_public
      });

      // 3. Trigger our secure Garbage Collector to wipe old files from S3 and PostgreSQL
      await api.delete(`/problems/${id}/testcases`);

      // 4. Send the single atomic batch
      if (testCases.length > 0) {
        await api.post(`/problems/${id}/testcases/batch`, formData, {
          headers: { 'Content-Type': 'multipart/form-data' }
        });
      }

      setStatus({ type: 'success', message: 'Changes saved and streamed to MinIO! 🎉' });
      setTimeout(() => navigate(`/arena/${id}`), 1500);

    } catch (error) {
      console.error("Critical Upload Error:", error);
      
      // 👇 COMPENSATING ACTION: Force draft state and alert
      try {
        await api.put(`/problems/${id}`, { 
          title: problemData.title,
          description: problemData.description,
          difficulty: problemData.difficulty,
          time_limit: problemData.time_limit,
          time_limit_ms: problemData.time_limit,
          memory_limit: problemData.memory_limit * 1024,
          memory_limit_kb: problemData.memory_limit * 1024,
          is_public: false // 👈 Force it to be a private draft
        });
        
        setStatus({ 
          type: 'error', 
          message: 'Network error during batch upload! The problem has been forced into a private Draft without test cases to prevent students from accessing a broken problem.' 
        });
      } catch (fallbackError) {
        setStatus({ type: 'error', message: 'CRITICAL ERROR: Upload failed and automatic draft fallback failed.' });
      }
    } finally {
      setIsSubmitting(false); 
    }
  };

  const handleDelete = async () => {
    if (!window.confirm("Are you absolute sure? This will delete the problem and ALL associated student submissions. This cannot be undone.")) return;
    
    setIsDeleting(true);
    try {
      await api.delete(`/problems/${id}`);
      navigate('/'); 
    } catch (err) {
      setStatus({ type: 'error', message: err.response?.data?.error || 'Failed to delete problem.' });
      setIsDeleting(false);
    }
  };

  if (isLoading) return <div className="flex justify-center items-center h-[calc(100vh-61px)] bg-dark-bg text-white text-xl">Loading Problem Data...</div>;

  return (
    <div className="h-[calc(100vh-61px)] bg-dark-bg text-gray-300 flex flex-col overflow-hidden relative">
      
      {status.message && (
        <div className={`p-3 text-center font-bold text-sm ${
          status.type === 'error' ? 'bg-red-900/90 text-red-200' :
          status.type === 'success' ? 'bg-green-900/90 text-green-200' :
          'bg-blue-900/90 text-blue-200'
        }`}>
          {status.message}
        </div>
      )}

      {step === 1 && (
        <div className="flex flex-1 h-full overflow-hidden">
          <div className="w-1/2 flex flex-col p-6 overflow-y-auto custom-scrollbar border-r border-dark-border bg-[#1e1e1e]">
            <h2 className="text-2xl font-bold text-white mb-6 tracking-wide">Edit Problem</h2>
            <div className="space-y-5 flex-1 flex flex-col">
              <div>
                <label className="text-xs font-bold text-gray-400 uppercase tracking-widest">Problem Name</label>
                <input required type="text" className="w-full bg-dark-bg text-white p-2.5 rounded border border-dark-border mt-1 outline-none focus:border-dark-accent transition-colors" 
                  value={problemData.title} onChange={e => setProblemData({...problemData, title: e.target.value})} />
              </div>
              <div className="flex space-x-4">
                <div className="w-1/3">
                  <label className="text-xs font-bold text-gray-400 uppercase tracking-widest">Difficulty</label>
                  <select className="w-full bg-dark-bg text-white p-2.5 rounded border border-dark-border mt-1 outline-none focus:border-dark-accent"
                    value={problemData.difficulty} onChange={e => setProblemData({...problemData, difficulty: e.target.value})}>
                    <option>Easy</option><option>Medium</option><option>Hard</option>
                  </select>
                </div>
                <div className="w-1/3">
                  <label className="text-xs font-bold text-gray-400 uppercase tracking-widest">Time Limit (ms)</label>
                  <input required type="number" step="100" min="500" max="5000" className="w-full bg-dark-bg text-white p-2.5 rounded border border-dark-border mt-1 outline-none focus:border-dark-accent"
                    value={problemData.time_limit} onChange={e => setProblemData({...problemData, time_limit: parseInt(e.target.value)})} />
                </div>
                <div className="w-1/3">
                  <label className="text-xs font-bold text-gray-400 uppercase tracking-widest">Memory (MB)</label>
                  <input required type="number" step="64" min="64" max="1024" className="w-full bg-dark-bg text-white p-2.5 rounded border border-dark-border mt-1 outline-none focus:border-dark-accent"
                    value={problemData.memory_limit} onChange={e => setProblemData({...problemData, memory_limit: parseInt(e.target.value)})} />
                </div>
              </div>

              <div className="flex-1 flex flex-col mt-4">
                <label className="text-xs font-bold text-gray-400 uppercase tracking-widest flex justify-between mb-1">
                  <span>Description (Markdown + LaTeX)</span>
                </label>
                <textarea required className="w-full flex-1 min-h-[300px] bg-dark-bg text-gray-300 p-4 rounded border border-dark-border font-mono text-sm outline-none focus:border-dark-accent resize-none custom-scrollbar"
                  value={problemData.description} onChange={e => setProblemData({...problemData, description: e.target.value})} />
              </div>

              <div className="flex items-center space-x-3 bg-[#121212] p-3 rounded border border-dark-border mt-4">
                <label className="text-xs font-bold text-gray-400 uppercase tracking-widest flex-1">Visibility Status</label>
                <div className="flex bg-[#1e1e1e] rounded p-1 border border-dark-border">
                  <button onClick={() => setProblemData({...problemData, is_public: false})}
                    className={`px-4 py-1.5 text-xs font-bold rounded transition-colors ${!problemData.is_public ? 'bg-yellow-900/30 text-yellow-500' : 'text-gray-500 hover:text-white'}`}>
                    DRAFT (Hidden)
                  </button>
                  <button onClick={() => setProblemData({...problemData, is_public: true})}
                    className={`px-4 py-1.5 text-xs font-bold rounded transition-colors ${problemData.is_public ? 'bg-green-900/30 text-green-500' : 'text-gray-500 hover:text-white'}`}>
                    PUBLIC (Live)
                  </button>
                </div>
              </div>

              <div className="flex justify-between items-center pt-4 border-t border-dark-border">
                {(currentUser?.role === 'admin' || (currentUser?.role === 'professor' && !problemData.is_public)) ? (
                  <button onClick={handleDelete} disabled={isDeleting} className="text-red-500 hover:text-red-400 text-sm font-bold flex items-center space-x-2 transition-colors">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                    <span>{isDeleting ? 'Erasing...' : 'Delete Problem'}</span>
                  </button>
                ) : <div />}
                <Button onClick={() => setStep(2)} variant="primary" disabled={!problemData.title || !problemData.description}>
                  Edit Test Cases →
                </Button>
              </div>
            </div>
          </div>

          <div className="w-1/2 bg-dark-bg p-8 overflow-y-auto custom-scrollbar">
             <div className="text-[10px] font-black text-gray-600 uppercase tracking-widest mb-6 border-b border-gray-800 pb-2">Arena Live Preview</div>
             <h2 className="text-3xl font-bold mb-3 text-white tracking-tight">{problemData.title || 'Untitled Problem'}</h2>
              <div className="flex flex-wrap gap-3 mb-6">
                <span className="bg-[#1e1e1e] text-gray-400 px-3 py-1 rounded text-xs border border-dark-border shadow-sm flex items-center gap-1.5">
                  ⏱️ {(problemData.time_limit || 2000) / 1000}s
                </span>
                <span className="bg-[#1e1e1e] text-gray-400 px-3 py-1 rounded text-xs border border-dark-border shadow-sm flex items-center gap-1.5">
                  💾 {problemData.memory_limit || 256}MB
                </span>
                <span className={`px-3 py-1 text-xs rounded font-bold border shadow-sm ${
                    problemData.difficulty === 'Easy' ? 'border-green-800 bg-green-900/20 text-green-400' : 
                    problemData.difficulty === 'Medium' ? 'border-yellow-800 bg-yellow-900/20 text-yellow-400' : 
                    'border-red-800 bg-red-900/20 text-red-400'
                  }`}>
                  {problemData.difficulty}
                </span>
              </div>
              <div className="prose prose-invert max-w-none text-gray-300 mb-8 text-[15px] leading-relaxed">
                <ReactMarkdown
                  remarkPlugins={[remarkMath, remarkGfm]}
                  rehypePlugins={[rehypeKatex]}
                  components={markdownComponents}
                >
                  {problemData.description}
                </ReactMarkdown>
              </div>
          </div>
        </div>
      )}

      {step === 2 && (
        <div className="flex-1 p-8 overflow-y-auto bg-dark-bg flex justify-center custom-scrollbar">
          <div className="w-full max-w-5xl">
            <div className="flex justify-between items-center mb-8 border-b border-dark-border pb-4">
              <div>
                <h2 className="text-3xl font-bold text-white tracking-wide">Test Case Arsenal</h2>
                <p className="text-gray-400 text-sm mt-1">Configure pure-text I/O pairs. These are sent directly to the Execution Engine memory.</p>
              </div>
              <Button onClick={() => setStep(1)} variant="secondary" size="sm">← Edit Problem Details</Button>
            </div>

            <div className="space-y-3">
              {testCases.map((tc, index) => (
                <div key={index} className="border border-dark-border rounded-lg overflow-hidden bg-[#1e1e1e] shadow-lg">
                  
                  {/* ACCORDION HEADER */}
                  <div 
                    className="flex justify-between items-center p-3 bg-[#2a2a2a] cursor-pointer hover:bg-[#333] transition-colors"
                    onClick={() => toggleTestCase(index)}
                  >
                    <div className="flex items-center space-x-3">
                      <span className="text-gray-400 font-mono text-xs w-4">
                        {expandedCases[index] ? '▼' : '▶'}
                      </span>
                      <span className="font-bold text-gray-200">Test Case {index + 1}</span>
                    </div>

                    <div className="flex items-center space-x-4" onClick={(e) => e.stopPropagation()}>
                      <label className="flex items-center space-x-2 text-sm text-gray-300 cursor-pointer">
                        <input 
                          type="checkbox" 
                          checked={tc.isSample}
                          onChange={(e) => updateTestCase(index, 'isSample', e.target.checked)}
                          className="rounded border-gray-600 bg-[#121212] text-green-500 focus:ring-green-500 focus:ring-offset-[#1e1e1e]"
                        />
                        <span className="select-none">Sample</span>
                      </label>
                      <label className="flex items-center space-x-2 text-sm text-gray-300 cursor-pointer">
                        <input 
                          type="checkbox" 
                          checked={tc.isHidden}
                          onChange={(e) => updateTestCase(index, 'isHidden', e.target.checked)}
                          className="rounded border-gray-600 bg-[#121212] text-blue-500 focus:ring-blue-500 focus:ring-offset-[#1e1e1e]"
                        />
                        <span className="select-none">Hidden</span>
                      </label>
                      <button 
                        type="button" 
                        onClick={() => handleRemoveTestCase(index)}
                        className="text-red-400 hover:text-white text-sm font-bold bg-red-900/20 hover:bg-red-600 px-3 py-1 rounded transition-colors"
                      >
                        Delete
                      </button>
                    </div>
                  </div>

                  {/* ACCORDION BODY */}
                  {expandedCases[index] && (
                    <div className="p-4 grid grid-cols-1 md:grid-cols-2 gap-4 bg-[#1a1a1a] border-t border-dark-border">
                      <div>
                        <label className="block text-[10px] font-black text-gray-500 uppercase tracking-widest mb-2">Input Data (stdin)</label>
                        <textarea
                          value={tc.input}
                          onChange={(e) => updateTestCase(index, 'input', e.target.value)}
                          rows="6"
                          className="w-full bg-[#121212] border border-dark-border rounded-lg p-3 text-sm text-gray-200 font-mono focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none transition-all resize-none custom-scrollbar"
                          placeholder="e.g. 5\n1 2 3 4 5"
                        />
                      </div>
                      <div>
                        <label className="block text-[10px] font-black text-gray-500 uppercase tracking-widest mb-2">Expected Output (stdout)</label>
                        <textarea
                          value={tc.expectedOutput}
                          onChange={(e) => updateTestCase(index, 'expectedOutput', e.target.value)}
                          rows="6"
                          className="w-full bg-[#121212] border border-dark-border rounded-lg p-3 text-sm text-green-400/90 font-mono focus:border-green-600 focus:ring-1 focus:ring-green-600 outline-none transition-all resize-none custom-scrollbar"
                          placeholder="e.g. 15"
                        />
                      </div>
                    </div>
                  )}
                  
                </div>
              ))}
            </div>

            {/* 👇 NEW: Drag and Drop zone wrapper */}
            <div 
              onDragOver={(e) => e.preventDefault()} 
              onDrop={handleZipUpload}
              className="mt-8 flex justify-between items-center bg-[#1e1e1e] p-5 rounded-lg border border-dark-border sticky bottom-4 shadow-2xl z-10 hover:border-blue-500 transition-colors"
            >
              <div className="flex space-x-3">
                <Button onClick={handleAddTestCase} variant="secondary" className="flex items-center space-x-2 border-dashed">
                  <span className="text-lg leading-none">+</span><span>Add Manually</span>
                </Button>
                <label className="flex items-center justify-center space-x-2 bg-[#2a2a2a] hover:bg-[#3a3a3a] text-gray-300 px-4 py-2 rounded text-sm font-bold border border-dark-border cursor-pointer transition-colors">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line></svg>
                  {/* 👇 NEW: Updated text for UI clarity */}
                  <span>Drop .zip or Click</span>
                  <input type="file" accept=".zip" className="hidden" onChange={handleZipUpload} />
                </label>
              </div>
              <Button onClick={handleSaveChanges} variant="success" className="px-8 shadow-lg shadow-green-900/20" 
                disabled={isSubmitting || testCases.some(tc => !tc.input?.trim() || !tc.expectedOutput?.trim())}>
                {isSubmitting ? 'Streaming to S3...' : 'Save All Changes'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}