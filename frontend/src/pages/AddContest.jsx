import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../services/api';
import Button from '../components/ui/Button';

export default function AddContest() {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [availableProblems, setAvailableProblems] = useState([]);
  const [assignedCache, setAssignedCache] = useState([]);

  const [fetchingProblems, setFetchingProblems] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 15;
  const [hasMore, setHasMore] = useState(true);

  // The master payload
  const [formData, setFormData] = useState({
    title: '',
    host_organization: 'CampusCompile Official',
    start_time: '',
    end_time: '',
    is_public: false,
    access_rules: {
      allowed_courses: [],
      allowed_departments: [],
      allowed_batches: [],
      allowed_sections: [],
      allowed_student_groups: [],
      allowed_graduation_years: []
    },
    problems: [] // Array of { problem_id, points_value }
  });

  const loadProblems = (currentOffset, query) => {
    setFetchingProblems(true);
    api.get('/faculty/problems', { params: { limit, offset: currentOffset, search: query } })
      .then(res => {
        const data = res.data?.problems || (Array.isArray(res.data) ? res.data : []);
        setAvailableProblems(data);
        setHasMore(data.length === limit);
        setOffset(currentOffset);
      })
      .catch(err => console.error("Failed to fetch problems", err))
      .finally(() => setFetchingProblems(false));
  };

  useEffect(() => {
    if (step === 3) {
      const timer = setTimeout(() => {
        setOffset(0);
        setHasMore(true);
        loadProblems(0, searchQuery);
      }, 400);
      return () => clearTimeout(timer);
    }
  }, [step, searchQuery]);

  const handlePrevProblems = () => {
    if (offset >= limit) {
      const nextOffset = offset - limit;
      loadProblems(nextOffset, searchQuery);
    }
  };

  const handleNextProblems = () => {
    if (hasMore) {
      const nextOffset = offset + limit;
      loadProblems(nextOffset, searchQuery);
    }
  };

  useEffect(() => {
    if (step !== 3) return;
    const handleKeyDown = (e) => {
      if (document.activeElement.tagName === 'INPUT' || document.activeElement.tagName === 'TEXTAREA') return;
      if (e.key === 'ArrowLeft') {
        handlePrevProblems();
      } else if (e.key === 'ArrowRight') {
        handleNextProblems();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [step, offset, hasMore, searchQuery]);

  const displayStart = availableProblems.length > 0 ? offset + 1 : 0;
  const displayEnd = offset + availableProblems.length;

  // Handle standard inputs
  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  // Handle comma-separated array inputs for access rules
  const handleArrayChange = (field, value) => {
    const arrayValue = value.split(',').map(item => item.trim()).filter(item => item !== '');

    // Graduation years need to be integers
    const finalArray = field === 'allowed_graduation_years'
      ? arrayValue.map(v => parseInt(v, 10)).filter(v => !isNaN(v))
      : arrayValue;

    setFormData(prev => ({
      ...prev,
      access_rules: {
        ...prev.access_rules,
        [field]: finalArray
      }
    }));
  };

  // Handle Problem Selection
  const toggleProblem = (prob) => {
    setFormData(prev => {
      const exists = prev.problems.find(p => p.problem_id === prob.problem_id);
      if (exists) {
        return { ...prev, problems: prev.problems.filter(p => p.problem_id !== prob.problem_id) };
      } else {
        return { ...prev, problems: [...prev.problems, { problem_id: prob.problem_id, points_value: 100 }] };
      }
    });

    setAssignedCache(prev => {
      if (!prev.find(p => p.problem_id === prob.problem_id)) {
        return [...prev, prob];
      }
      return prev;
    });
  };

  const updatePoints = (problemId, points) => {
    setFormData(prev => ({
      ...prev,
      problems: prev.problems.map(p =>
        p.problem_id === problemId ? { ...p, points_value: parseInt(points, 10) || 0 } : p
      )
    }));
  };

  const handleSubmit = async () => {
    setLoading(true);
    try {
      // Create a cloned payload and force the dates into full ISO strings for Go
      const payload = {
        ...formData,
        start_time: new Date(formData.start_time).toISOString(),
        end_time: new Date(formData.end_time).toISOString(),
      };

      // Send the formatted payload
      await api.post('/contests', payload);
      navigate('/contests');
    } catch (error) {
      // Log the EXACT error message the Go backend sends back
      console.error("Failed to create contest:", error.response?.data || error.message);

      // Show the exact Gin validation error in the alert
      const errorMsg = error.response?.data?.error || "Unknown Error. Check console.";
      alert(`Backend Error: ${errorMsg}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-8">
      <div className="mb-8 border-b border-dark-border pb-4 flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold text-white">Forge Contest</h2>
          <p className="text-gray-400 mt-1">Design your competition and set access rules.</p>
        </div>

        {/* Step Indicator */}
        <div className="flex gap-2">
          {[1, 2, 3].map(num => (
            <div key={num} className={`h-2 w-12 rounded-full ${step >= num ? 'bg-blue-500' : 'bg-gray-700'}`} />
          ))}
        </div>
      </div>

      <div className="bg-[#1e1e1e] border border-dark-border rounded-lg p-6 shadow-xl">

        {/* === STEP 1: Details === */}
        {step === 1 && (
          <div className="space-y-6 animate-fadeIn">
            <h3 className="text-xl font-bold text-white border-b border-dark-border pb-2">Step 1: Details & Timing</h3>

            <div className="grid grid-cols-2 gap-6">
              <div className="col-span-2">
                <label className="block text-gray-400 text-sm font-bold mb-2">Contest Title</label>
                <input type="text" name="title" value={formData.title} onChange={handleChange} placeholder="e.g., ICPC Prelims 2026" className="w-full p-3 rounded bg-dark-surface border border-dark-border text-white focus:outline-none focus:border-blue-500" required />
              </div>

              <div className="col-span-2">
                <label className="block text-gray-400 text-sm font-bold mb-2">Host Organization</label>
                <input type="text" name="host_organization" value={formData.host_organization} onChange={handleChange} placeholder="e.g., Bennett University" className="w-full p-3 rounded bg-dark-surface border border-dark-border text-white focus:outline-none focus:border-blue-500" />
              </div>

              <div>
                <label className="block text-gray-400 text-sm font-bold mb-2">Start Time</label>
                <input type="datetime-local" name="start_time" value={formData.start_time} onChange={handleChange} className="w-full p-3 rounded bg-dark-surface border border-dark-border text-white" required />
              </div>

              <div>
                <label className="block text-gray-400 text-sm font-bold mb-2">End Time</label>
                <input type="datetime-local" name="end_time" value={formData.end_time} onChange={handleChange} className="w-full p-3 rounded bg-dark-surface border border-dark-border text-white" required />
              </div>

              <div className="col-span-2 flex items-center gap-3 p-4 bg-[#2a2a2a] rounded border border-dark-border mt-2">
                <input type="checkbox" name="is_public" checked={formData.is_public} onChange={handleChange} className="w-5 h-5 accent-green-500" />
                <div>
                  <p className="text-white font-bold">Publish to Live Arena</p>
                  <p className="text-xs text-gray-400">If unchecked, this contest will remain a draft in your workspace.</p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* === STEP 2: Access Control === */}
        {step === 2 && (
          <div className="space-y-6 animate-fadeIn">
            <h3 className="text-xl font-bold text-white border-b border-dark-border pb-2">Step 2: Demographic Clearance</h3>
            <p className="text-sm text-gray-400 mb-4">Leave a field blank to allow everyone. Separate multiple entries with commas.</p>

            <div className="grid grid-cols-2 gap-6">
              {[
                { label: 'Allowed Courses', field: 'allowed_courses', placeholder: 'e.g., B.Tech, M.Tech' },
                { label: 'Allowed Departments', field: 'allowed_departments', placeholder: 'e.g., CSE, ECE' },
                { label: 'Allowed Batches', field: 'allowed_batches', placeholder: 'e.g., 2024-2028' },
                { label: 'Allowed Sections', field: 'allowed_sections', placeholder: 'e.g., A, B, C' },
                { label: 'Allowed Student Groups', field: 'allowed_student_groups', placeholder: 'e.g., G1, G2' },
                { label: 'Allowed Graduation Years', field: 'allowed_graduation_years', placeholder: 'e.g., 2028, 2029' },
              ].map(item => (
                <div key={item.field}>
                  <label className="block text-gray-400 text-sm font-bold mb-2">{item.label}</label>
                  <input
                    type="text"
                    defaultValue={formData.access_rules[item.field].join(', ')}
                    onChange={(e) => handleArrayChange(item.field, e.target.value)}
                    placeholder={item.placeholder}
                    className="w-full p-3 rounded bg-dark-surface border border-dark-border text-white focus:border-blue-500 outline-none font-mono text-sm"
                  />
                </div>
              ))}
            </div>
          </div>
        )}

        {/* === STEP 3: Problem Selection === */}
        {step === 3 && (
          <div className="space-y-6 animate-fadeIn">
            <h3 className="text-xl font-bold text-white border-b border-dark-border pb-2">Step 3: Arena Setup</h3>
            <p className="text-sm text-gray-400 mb-4">Select the problems you want to feature in this contest and assign point values.</p>

            {/* Pinned Selected Problems */}
            {formData.problems.length > 0 && (
              <div className="mb-6">
                <h4 className="text-sm font-bold text-gray-400 mb-2">Selected Problems</h4>
                <div className="space-y-2 max-h-64 overflow-y-auto pr-2">
                  {formData.problems.map(selectedData => {
                    const prob = assignedCache.find(p => p.problem_id === selectedData.problem_id) || 
                                 availableProblems.find(p => p.problem_id === selectedData.problem_id) || 
                                 { problem_id: selectedData.problem_id, title: 'Pinned Problem', difficulty: 'Unknown' };
                    return (
                      <div key={prob.problem_id} className="flex items-center justify-between p-3 rounded border border-blue-500 bg-blue-900/10 transition">
                        <div className="flex items-center gap-4">
                          <input type="checkbox" checked={true} onChange={() => toggleProblem(prob)} className="w-5 h-5 accent-blue-500" />
                          <div>
                            <p className="text-white font-bold text-sm">{prob.title}</p>
                            <p className={`text-xs ${prob.difficulty === 'Easy' ? 'text-green-400' : prob.difficulty === 'Medium' ? 'text-yellow-400' : 'text-red-400'}`}>{prob.difficulty}</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <label className="text-xs text-gray-400 font-bold">Points:</label>
                          <input type="number" value={selectedData.points_value || 100} onChange={(e) => updatePoints(prob.problem_id, e.target.value)} className="w-16 p-1 rounded bg-[#1e1e1e] border border-dark-border text-white text-center font-mono text-sm" />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            <div className="mb-4">
              <input
                type="text"
                placeholder="Search your workspace..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full p-3 rounded bg-dark-surface border border-dark-border text-white focus:outline-none focus:border-blue-500 transition shadow-sm text-sm"
              />
            </div>

            <div className="max-h-96 overflow-y-auto pr-2 space-y-3">
              {fetchingProblems && offset === 0 ? (
                <p className="text-center text-gray-400 py-10 animate-pulse">Loading workspace...</p>
              ) : availableProblems.length === 0 ? (
                <p className="text-center text-gray-500 italic py-10">No problems found matching this search in your workspace. You need to forge problems first!</p>
              ) : (
                <>
                  {availableProblems.filter(prob => !formData.problems.some(p => p.problem_id === prob.problem_id)).map(prob => {
                    return (
                      <div key={prob.problem_id} className="flex items-center justify-between p-4 rounded border transition border-dark-border bg-dark-surface hover:bg-[#252525]">
                        <div className="flex items-center gap-4">
                          <input type="checkbox" checked={false} onChange={() => toggleProblem(prob)} className="w-5 h-5 accent-blue-500" />
                          <div>
                            <p className="text-white font-bold">{prob.title}</p>
                            <p className={`text-xs ${prob.difficulty === 'Easy' ? 'text-green-400' : prob.difficulty === 'Medium' ? 'text-yellow-400' : 'text-red-400'}`}>{prob.difficulty}</p>
                          </div>
                        </div>
                      </div>
                    )
                  })}

                  {!fetchingProblems && (hasMore || offset > 0) && (
                  <div className="flex justify-between items-center mt-2 pt-2 border-t border-dark-border px-2">
                    <Button onClick={handlePrevProblems} variant="outline" disabled={offset === 0} className="w-1/5 text-gray-300 border-dark-border hover:bg-[#2a2a2a] disabled:opacity-30 disabled:cursor-not-allowed text-xs py-1 px-3">
                      &larr; Prev
                    </Button>
                    <span className="w-3/5 text-center text-gray-400 text-xs font-mono">
                      Showing {displayStart} - {displayEnd}
                    </span>
                    <Button onClick={handleNextProblems} variant="outline" disabled={!hasMore} className="w-1/5 text-gray-300 border-dark-border hover:bg-[#2a2a2a] disabled:opacity-30 disabled:cursor-not-allowed text-xs py-1 px-3">
                      Next &rarr;
                    </Button>
                  </div>
                )}
                </>
              )}
            </div>
          </div>
        )}

        {/* Form Navigation Controls */}
        <div className="flex justify-between mt-8 pt-6 border-t border-dark-border">
          {step > 1 ? (
            <Button onClick={() => setStep(step - 1)} variant="secondary">Back</Button>
          ) : <div></div>}

          {step < 3 ? (
            <Button onClick={() => setStep(step + 1)} variant="primary">Next Step</Button>
          ) : (
            <Button onClick={handleSubmit} variant="success" disabled={loading} className="border border-green-600">
              {loading ? 'Initializing Arena...' : 'Launch Contest'}
            </Button>
          )}
        </div>

      </div>
    </div>
  );
}