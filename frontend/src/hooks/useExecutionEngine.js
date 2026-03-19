import { useState, useEffect, useRef } from 'react';
import api from '../services/api';
import { useAuth } from '../context/AuthContext';

export const boilerplates = {
  cpp: `#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n    // Write your C++ code here\n    return 0;\n}`,
  python: `# Write your Python code here`,
  java: `import java.util.*;\nimport java.io.*;\n\n public class Main {\n    public static void main(String[] args) {\n        // Write your Java code here\n    }\n}`
};

export default function useExecutionEngine(problemId, contestId = null) {
  const { currentUser } = useAuth();
  
  // UI and Editor State
  const [problem, setProblem] = useState(null);
  const [code, setCode] = useState(boilerplates['cpp']);
  const [language, setLanguage] = useState('cpp');
  const [submitStatus, setSubmitStatus] = useState(''); 
  const [history, setHistory] = useState([]);
  const [historyOffset, setHistoryOffset] = useState(0);
  const historyLimit = 10;
  const [hasMoreHistory, setHasMoreHistory] = useState(true);
  
  // Console State
  const [isConsoleOpen, setIsConsoleOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('input');
  const [customInput, setCustomInput] = useState('');
  const [consoleOutput, setConsoleOutput] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);

  const sseRef = useRef(null);

  // Dynamic Draft Key
  const draftKey = currentUser ? (
    contestId 
      ? `draft_contest_${contestId}_${currentUser.id}_${problemId}` 
      : `draft_${currentUser.id}_${problemId}`
  ) : null;

  // Cleanup SSE on unmount
  useEffect(() => {
    return () => { if (sseRef.current) sseRef.current.close(); };
  }, []);

  // Fetch Problem Details & History
  useEffect(() => {
    if (!problemId) return;
    
    api.get(`/problems/${problemId}`)
      .then(res => setProblem(res.data))
      .catch(err => console.error("Could not fetch problem details", err));
      
    fetchHistory(0);
  }, [problemId, contestId]);

  // Load Draft
  useEffect(() => {
    if (draftKey) {
      const savedDraft = localStorage.getItem(draftKey);
      if (savedDraft) {
        try {
          const parsed = JSON.parse(savedDraft);
          setLanguage(parsed.language);
          setCode(parsed.code);
          return; 
        } catch (e) {
          console.error("Failed to parse draft", e);
        }
      }
    }
    setCode(boilerplates['cpp']);
  }, [problemId, draftKey]);

  // Save Draft (Debounced)
  useEffect(() => {
    if (!draftKey || !code) return;
    const timer = setTimeout(() => {
      if (code !== boilerplates[language]) {
         localStorage.setItem(draftKey, JSON.stringify({ language, code }));
      }
    }, 1000); 
    return () => clearTimeout(timer);
  }, [code, language, draftKey]);

  const fetchHistory = async (offset = 0) => {
    try {
      const endpoint = contestId 
        ? `/submissions/history/${problemId}?contest_id=${contestId}&limit=${historyLimit}&offset=${offset}` 
        : `/submissions/history/${problemId}?limit=${historyLimit}&offset=${offset}`;
      const res = await api.get(endpoint);
      const data = res.data || [];
      if (offset === 0) setHistory(data);
      else setHistory(prev => [...prev, ...data]);
      setHasMoreHistory(data.length === historyLimit);
      setHistoryOffset(offset);
    } catch (err) {
      console.error("Could not fetch history:", err);
    }
  };

  const loadMoreHistory = () => fetchHistory(historyOffset + historyLimit);

  const handleSubmit = async () => {
    setIsProcessing(true);
    setSubmitStatus('Pending... ⏳');
    setIsConsoleOpen(false);
    if (sseRef.current) sseRef.current.close();

    try {
      const payload = { problem_id: problemId, language, source_code: code };
      if (contestId) payload.contest_id = contestId;

      const response = await api.post('/submit', payload);
      const token = localStorage.getItem('token');
      
      sseRef.current = new EventSource(`${api.defaults.baseURL}/submissions/stream/${response.data.submission_id}?token=${token}`);
      
      sseRef.current.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.status === 'Running') {
          setSubmitStatus('Running... ⚙️');
        } else {
          setSubmitStatus(data.status);
          fetchHistory(0);
          
          if (['CE', 'RE', 'WA', 'TLE', 'SE'].includes(data.status)) {
            setConsoleOutput(data.message || `Verdict: ${data.status}`);
            setActiveTab('output');
            setIsConsoleOpen(true);
          } else if (data.status === 'AC' || data.status === 'Accepted') {
            setConsoleOutput("Execution Successful! 🎉\nAll test cases passed.");
            setActiveTab('output');
            setIsConsoleOpen(true);
            if (draftKey) localStorage.removeItem(draftKey);
          }
          setIsProcessing(false);
          sseRef.current.close();
        }
      };

      sseRef.current.onerror = () => { 
        setIsProcessing(false);
        sseRef.current.close(); 
      };
    } catch (error) {
      const errorMessage = error.customMessage || 'Error: Submission Failed';
      const status = error.response?.status;
      
      // Dynamically assign the visual status based on the actual HTTP code
      let finalStatus = 'Error ❌';
      if (status === 429) finalStatus = 'Rate Limited 🛑';
      else if (status === 503) finalStatus = 'Unavailable ⚠️';
      
      setSubmitStatus(finalStatus);
      setConsoleOutput(`[SYSTEM REJECTED]\n${errorMessage}`);
      setActiveTab('output');
      setIsConsoleOpen(true);
      setIsProcessing(false);
    }
  };

  const handleRunCode = async () => {
    setIsProcessing(true);
    setConsoleOutput('Queuing... ⚙️');
    setActiveTab('output');
    setIsConsoleOpen(true);
    if (sseRef.current) sseRef.current.close();

    try {
      const response = await api.post('/run', { language, source_code: code, custom_input: customInput });
      const token = localStorage.getItem('token');
      
      sseRef.current = new EventSource(`${api.defaults.baseURL}/run/stream/${response.data.run_id}?token=${token}`);
      
      sseRef.current.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.status === 'Running') {
          setConsoleOutput('Running... ⚙️');
        } else if (['Completed', 'CE', 'RE', 'TLE', 'SE'].includes(data.status)) {
          setConsoleOutput(data.output || data.message || "Program finished with no output.");
          setIsProcessing(false);
          sseRef.current.close();
        }
      };

      sseRef.current.onerror = () => {
        setConsoleOutput('Error streaming execution status.');
        setIsProcessing(false);
        sseRef.current.close();
      };
    } catch (error) {
      const errorMessage = error.customMessage || 'Error: Could not connect to execution engine.';
      
      // Keep the console output descriptive without falsely claiming they were "blocked"
      setConsoleOutput(`[SYSTEM REJECTED]\n${errorMessage}`);
      setActiveTab('output');
      setIsConsoleOpen(true);
      setIsProcessing(false);
    }
  };

  return {
    problem, code, setCode, language, setLanguage, submitStatus, history,
    hasMoreHistory, loadMoreHistory,
    isConsoleOpen, setIsConsoleOpen, activeTab, setActiveTab,
    customInput, setCustomInput, consoleOutput, isProcessing,
    handleSubmit, handleRunCode
  };
}