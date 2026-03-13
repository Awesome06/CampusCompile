import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import JSZip from 'jszip';
import api from '../services/api'; 
import Button from '../components/ui/Button';

const DEFAULT_DESCRIPTION = `### Problem Statement
Write your problem statement here. CampusCompile supports inline math like $O(N \\log N)$ and block equations:
$$ \\sum_{i=1}^{n} i = \\frac{n(n+1)}{2} $$

### Constraints
* $1 \\le N \\le 10^5$
* $1 \\le A_i \\le 10^9$

### Sample Input
\`\`\`text
5
1 2 3 4 5
\`\`\`

### Sample Output
\`\`\`text
15
\`\`\`
`;

export default function AddProblem() {
  // CodeChef-Style Custom Renderers for the Live Preview
  const markdownComponents = {
    h3: ({node, ...props}) => <h3 className="text-xl font-bold text-blue-400 mt-8 mb-4 border-b border-dark-border pb-2 tracking-wide uppercase" {...props} />,
    pre: ({node, ...props}) => <pre className="bg-[#121212] border border-dark-border rounded-lg p-5 overflow-x-auto my-4 font-mono text-sm text-gray-300 shadow-inner" {...props} />,
    code: ({node, inline, ...props}) => inline 
        ? <code className="bg-[#2a2a2a] text-pink-400 px-1.5 py-0.5 rounded text-sm font-mono border border-dark-border" {...props} /> 
        : <code {...props} />,
    ul: ({node, ...props}) => <ul className="list-disc list-inside my-4 space-y-2 text-gray-300 marker:text-blue-500" {...props} />,
    li: ({node, ...props}) => <li className="leading-relaxed" {...props} />,
    p: ({node, ...props}) => <p className="my-4 leading-relaxed text-gray-300" {...props} />,
    blockquote: ({node, ...props}) => <blockquote className="border-l-4 border-blue-500 bg-blue-900/10 p-4 my-4 rounded-r-lg italic text-gray-400" {...props} />
  };

  const navigate = useNavigate();
  const [isAuthorized, setIsAuthorized] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  // Wizard & Submission State
  const [step, setStep] = useState(1);
  const [status, setStatus] = useState({ type: '', message: '' });
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Modal State
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [publishedUrl, setPublishedUrl] = useState('');
  const [copied, setCopied] = useState(false);

  // Step 1: Problem Details
  const [problemData, setProblemData] = useState({
    title: '',
    description: DEFAULT_DESCRIPTION,
    difficulty: 'Easy',
    time_limit: 2000,
    memory_limit: 256,
    is_public: false
  });

  // Step 2: Dynamic Test Cases
  const [testCases, setTestCases] = useState([
    { input: '', expectedOutput: '', isHidden: false }
  ]);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) {
      navigate('/login');
      return;
    }
    try {
      const decodedToken = jwtDecode(token);
      const userRole = decodedToken.role?.toLowerCase(); 
      setIsAuthorized(userRole === 'admin' || userRole === 'professor');

    } catch (error) {
      setIsAuthorized(false);
    } finally {
      setIsLoading(false);
    }
  }, [navigate]);

  const handleAddTestCase = () => setTestCases([...testCases, { input: '', expectedOutput: '', isHidden: true }]);
  const handleRemoveTestCase = (index) => setTestCases(testCases.filter((_, i) => i !== index));
  const updateTestCase = (index, field, value) => {
    const updated = [...testCases];
    updated[index][field] = value;
    setTestCases(updated);
  };

  const handleZipUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    
    setStatus({ type: 'info', message: 'Extracting test cases from ZIP...' });
    
    try {
      const zip = new JSZip();
      const loadedZip = await zip.loadAsync(file);
      
      const inputs = {};
      const outputs = {};
      
      // 1. Read all files in the ZIP
      for (const [relativePath, zipEntry] of Object.entries(loadedZip.files)) {
        if (zipEntry.dir) continue; // Skip folders
        
        const content = await zipEntry.async("string");
        const cleanName = relativePath.split('/').pop().toLowerCase();
        
        // Extract the base number/name (e.g., "1.in" -> "1", "input_5.txt" -> "5")
        const baseMatch = cleanName.match(/(\d+)/); 
        const baseName = baseMatch ? baseMatch[0] : cleanName.split('.')[0];
        
        if (cleanName.includes('in')) inputs[baseName] = content;
        if (cleanName.includes('out')) outputs[baseName] = content;
      }

      // 2. Pair them up
      const newTestCases = [];
      Object.keys(inputs).forEach(key => {
        if (outputs[key]) {
          newTestCases.push({
            input: inputs[key].trim(),
            expectedOutput: outputs[key].trim(),
            isHidden: true // Bulk uploads default to hidden
          });
        }
      });

      if (newTestCases.length === 0) {
        setStatus({ type: 'error', message: 'No matching input/output files found in ZIP. Ensure files have "in" and "out" in their names.' });
        return;
      }

      // 3. Append to existing state (PREVENTS OVERWRITES)
      setTestCases(prev => [...prev, ...newTestCases]);
      setStatus({ type: 'success', message: `Successfully appended ${newTestCases.length} test cases!` });
      setTimeout(() => setStatus({ type: '', message: '' }), 3000);

    } catch (err) {
      console.error(err);
      setStatus({ type: 'error', message: 'Failed to process ZIP file.' });
    }
    // Reset input so they can upload another zip if needed
    e.target.value = null; 
  };

  const handlePublish = async () => {
    setIsSubmitting(true);
    setStatus({ type: 'info', message: 'Forging problem and syncing test cases... 🚀' });

    try {
      const problemPayload = {
        title: problemData.title,
        description: problemData.description,
        difficulty: problemData.difficulty,
        time_limit: problemData.time_limit,
        memory_limit: problemData.memory_limit * 1024,
        is_public: problemData.is_public
      };
      
      const probRes = await api.post('/problems', problemPayload);
      const newProblemId = probRes.data.problem_id;

      await api.post(`/problems/${newProblemId}/testcases/batch`, {
        test_cases: testCases
      });

      setStatus({ type: 'success', message: 'Problem published successfully! 🎉' });
      
      // Construct the full URL for sharing and trigger the modal
      const fullUrl = `${window.location.origin}/arena/${newProblemId}`;
      setPublishedUrl(fullUrl);
      setShowSuccessModal(true);

    } catch (err) {
      setStatus({ type: 'error', message: err.response?.data?.error || 'Failed to publish problem.' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCopyUrl = () => {
    navigator.clipboard.writeText(publishedUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const resetForm = () => {
    setProblemData({ title: '', description: DEFAULT_DESCRIPTION, difficulty: 'Easy', time_limit: 2000, memory_limit: 256 });
    setTestCases([{ input: '', expectedOutput: '', isHidden: false }]);
    setStatus({ type: '', message: '' });
    setShowSuccessModal(false);
    setStep(1);
  };

  if (isLoading) return <div className="flex justify-center items-center h-[calc(100vh-61px)] bg-dark-bg text-white text-xl">Verifying clearance...</div>;

  if (!isAuthorized) {
    return (
      <div className="flex justify-center items-center h-[calc(100vh-61px)] bg-dark-bg">
        {/* ... Access Denied UI remains the same ... */}
      </div>
    );
  }

  return (
    <div className="h-[calc(100vh-61px)] bg-dark-bg text-gray-300 flex flex-col overflow-hidden relative">
      
      {/* Top Status Bar */}
      {status.message && !showSuccessModal && (
        <div className={`p-3 text-center font-bold text-sm ${
          status.type === 'error' ? 'bg-red-900/90 text-red-200' :
          status.type === 'success' ? 'bg-green-900/90 text-green-200' :
          'bg-blue-900/90 text-blue-200'
        }`}>
          {status.message}
        </div>
      )}

      {/* --- STEP 1: SPLIT PANE EDITOR --- */}
      {step === 1 && (
        <div className="flex flex-1 h-full overflow-hidden">
          {/* Left Pane: Configuration Form */}
          <div className="w-1/2 flex flex-col p-6 overflow-y-auto custom-scrollbar border-r border-dark-border bg-[#1e1e1e]">
            <h2 className="text-2xl font-bold text-white mb-6 tracking-wide">Forge New Problem</h2>
            <div className="space-y-5 flex-1 flex flex-col">
              <div>
                <label className="text-xs font-bold text-gray-400 uppercase tracking-widest">Problem Name</label>
                <input required type="text" placeholder="e.g., Two Sum" className="w-full bg-dark-bg text-white p-2.5 rounded border border-dark-border mt-1 outline-none focus:border-dark-accent transition-colors" 
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
                  <a href="https://katex.org/docs/supported.html" target="_blank" rel="noreferrer" className="text-dark-accent hover:underline">Math Guide</a>
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
              <div className="flex justify-end pt-4 border-t border-dark-border">
                <Button onClick={() => setStep(2)} variant="primary" disabled={!problemData.title || !problemData.description}>
                  Next: Setup Test Cases →
                </Button>
              </div>
            </div>
          </div>

          {/* Right Pane: Arena Live Preview */}
          <div className="w-1/2 bg-dark-bg p-8 overflow-y-auto custom-scrollbar">
             <div className="text-[10px] font-black text-gray-600 uppercase tracking-widest mb-6 border-b border-gray-800 pb-2">Arena Live Preview</div>
             <h2 className="text-3xl font-bold mb-3 text-white tracking-tight">{problemData.title || 'Untitled Problem'}</h2>
              <div className="flex space-x-3 mb-6">
                <span className="bg-[#1e1e1e] text-gray-400 px-3 py-1 rounded text-xs border border-dark-border shadow-sm">⏱️ {problemData.time_limit}ms</span>
                <span className="bg-[#1e1e1e] text-gray-400 px-3 py-1 rounded text-xs border border-dark-border shadow-sm">💾 {problemData.memory_limit}MB</span>
                <span className={`px-3 py-1 text-xs rounded font-bold border shadow-sm ${
                  problemData.difficulty === 'Easy' ? 'border-green-800 bg-green-900/20 text-green-400' : 
                  problemData.difficulty === 'Medium' ? 'border-yellow-800 bg-yellow-900/20 text-yellow-400' : 'border-red-800 bg-red-900/20 text-red-400'
                }`}>{problemData.difficulty}</span>
              </div>
              <div className="prose prose-invert max-w-none text-[15px] leading-relaxed">
              <ReactMarkdown 
                remarkPlugins={[remarkMath, remarkGfm]} 
                rehypePlugins={[rehypeKatex]}
                components={markdownComponents}
              >
                {problemData.description || '*Preview your problem description here...*'}
              </ReactMarkdown>
            </div>
          </div>
        </div>
      )}

      {/* --- STEP 2: TEST CASE CONFIGURATOR --- */}
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

            <div className="space-y-6">
              {testCases.map((tc, index) => (
                <div key={index} className="bg-[#1e1e1e] border border-dark-border rounded-lg p-5 relative shadow-xl transition-all hover:border-gray-600">
                  <div className="flex justify-between items-center mb-4 border-b border-dark-border/50 pb-3">
                    <h3 className="text-white font-bold tracking-widest text-sm uppercase flex items-center space-x-2">
                      <span className="bg-dark-accent text-white px-2 py-0.5 rounded text-[10px]">#{index + 1}</span>
                      <span>Test Case</span>
                    </h3>
                    <div className="flex items-center space-x-5">
                      <label className="flex items-center space-x-2 text-sm text-gray-400 cursor-pointer hover:text-white transition-colors">
                        <input type="checkbox" checked={tc.isHidden} onChange={(e) => updateTestCase(index, 'isHidden', e.target.checked)} 
                          className="w-4 h-4 rounded bg-dark-bg border-dark-border text-dark-accent focus:ring-dark-accent focus:ring-offset-dark-bg" />
                        <span className="select-none">Hidden Evaluation Case</span>
                      </label>
                      {testCases.length > 1 && (
                        <button onClick={() => handleRemoveTestCase(index)} className="text-red-500 hover:text-red-400 opacity-70 hover:opacity-100 transition-opacity" title="Remove Test Case">
                          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                        </button>
                      )}
                    </div>
                  </div>
                  <div className="flex space-x-6">
                    <div className="w-1/2 flex flex-col">
                      <label className="text-[10px] font-black text-gray-500 uppercase tracking-widest mb-2">Input Data (stdin)</label>
                      <textarea required className="w-full h-40 bg-[#121212] text-gray-300 p-3 rounded border border-dark-border font-mono text-sm outline-none focus:border-dark-accent resize-none custom-scrollbar"
                        value={tc.input} onChange={(e) => updateTestCase(index, 'input', e.target.value)} placeholder="e.g.,\n5\n1 2 3 4 5" />
                    </div>
                    <div className="w-1/2 flex flex-col">
                      <label className="text-[10px] font-black text-gray-500 uppercase tracking-widest mb-2">Expected Output (stdout)</label>
                      <textarea required className="w-full h-40 bg-[#121212] text-green-400/90 p-3 rounded border border-dark-border font-mono text-sm outline-none focus:border-green-600 resize-none custom-scrollbar"
                        value={tc.expectedOutput} onChange={(e) => updateTestCase(index, 'expectedOutput', e.target.value)} placeholder="e.g.,\n15" />
                    </div>
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-8 flex justify-between items-center bg-[#1e1e1e] p-5 rounded-lg border border-dark-border sticky bottom-4 shadow-2xl z-10">
              <div className="flex space-x-3">
                <Button onClick={handleAddTestCase} variant="secondary" className="flex items-center space-x-2 border-dashed">
                  <span className="text-lg leading-none">+</span><span>Add Manually</span>
                </Button>
                
                {/* NEW: ZIP Upload Button */}
                <label className="flex items-center justify-center space-x-2 bg-[#2a2a2a] hover:bg-[#3a3a3a] text-gray-300 px-4 py-2 rounded text-sm font-bold border border-dark-border cursor-pointer transition-colors">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line></svg>
                  <span>Bulk Upload (.zip)</span>
                  <input type="file" accept=".zip" className="hidden" onChange={handleZipUpload} />
                </label>
              </div>

              <Button onClick={handlePublish} variant="success" className="px-8 shadow-lg shadow-green-900/20" 
                disabled={isSubmitting || testCases.some(tc => !tc.input.trim() || !tc.expectedOutput.trim())}>
                {isSubmitting ? 'Syncing to Database...' : 'Finalize & Publish'}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* --- SUCCESS MODAL --- */}
      {showSuccessModal && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 backdrop-blur-sm p-4">
          <div className="bg-[#1e1e1e] w-full max-w-md p-8 rounded-xl border border-dark-border shadow-2xl">
            <div className="text-center mb-6">
              <div className="w-16 h-16 bg-green-900/30 text-green-500 rounded-full flex items-center justify-center mx-auto mb-4 border border-green-800/50">
                <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
              </div>
              <h2 className="text-2xl font-bold text-white mb-2">Problem Forged!</h2>
              <p className="text-gray-400 text-sm">Your challenge is now live. Share this secure link with your students.</p>
            </div>

            <div className="flex items-center space-x-2 bg-dark-bg p-2 rounded border border-dark-border mb-8">
              <input type="text" readOnly value={publishedUrl} className="flex-1 bg-transparent text-gray-300 text-sm font-mono outline-none px-2 select-all" />
              <button 
                onClick={handleCopyUrl} 
                className={`p-2 rounded transition-colors ${copied ? 'bg-green-900/50 text-green-400' : 'bg-[#2a2a2a] text-gray-400 hover:text-white hover:bg-[#3a3a3a]'}`}
                title="Copy to Clipboard"
              >
                {copied ? (
                  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
                ) : (
                  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                )}
              </button>
            </div>

            <div className="flex space-x-4">
              <Button onClick={resetForm} variant="secondary" className="flex-1">Create Another</Button>
              <Button onClick={() => navigate(new URL(publishedUrl).pathname)} variant="success" className="flex-1">Go to Arena</Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}