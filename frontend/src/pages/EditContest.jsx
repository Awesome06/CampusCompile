import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import api from '../services/api';
import Button from '../components/ui/Button';

const defaultAccessRules = {
  allowed_courses: [],
  allowed_departments: [],
  allowed_batches: [],
  allowed_sections: [],
  allowed_student_groups: [],
  allowed_graduation_years: []
};

export default function EditContest() {
  const { id } = useParams();
  const navigate = useNavigate();
  const userRole = localStorage.getItem('role');
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [fetchingProblems, setFetchingProblems] = useState(false);
  const [availableProblems, setAvailableProblems] = useState([]);
  const [assignedCache, setAssignedCache] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 25;
  const [hasMore, setHasMore] = useState(true);

  const [formData, setFormData] = useState({
    title: '', host_organization: '', start_time: '', end_time: '',
    is_public: false, access_rules: defaultAccessRules, problems: []
  });

  const isTimeLocked = formData.start_time && new Date(formData.start_time) <= new Date() && userRole !== 'admin';

  // 👇 CRASH FIX 1: Safely handle "Invalid Date" objects before calling .toISOString()
  const formatForInput = (isoString) => {
    if (!isoString) return '';
    try {
      const d = new Date(isoString);
      if (isNaN(d.getTime())) return ''; // Prevents RangeError crash
      return d.toISOString().slice(0, 16);
    } catch (e) {
      console.warn("Date parsing error:", e);
      return '';
    }
  };

  const loadProblems = (currentOffset, query) => {
    setFetchingProblems(true);
    api.get('/faculty/problems', { params: { limit, offset: currentOffset, search: query } })
      .then(res => {
        const data = res.data || [];
        setAvailableProblems(data);
        setHasMore(data.length === limit);
        setOffset(currentOffset);
      })
      .catch(err => console.error("Failed to fetch problems", err))
      .finally(() => setFetchingProblems(false));
  };

  useEffect(() => {
    const loadContestData = async () => {
      try {
        const [contestRes, assignedProbsRes] = await Promise.all([
          api.get(`/contests/${id}`),
          api.get(`/contests/${id}/problems`)
        ]);

        const contest = contestRes.data?.contest || {};
        const assignedProblems = Array.isArray(assignedProbsRes.data) ? assignedProbsRes.data : [];

        setFormData({
          title: contest.title || '',
          host_organization: contest.host_organization || 'CampusCompile Official',
          start_time: formatForInput(contest.start_time),
          end_time: formatForInput(contest.end_time),
          is_public: contest.is_public || false,
          access_rules: contest.access_rules || defaultAccessRules,
          problems: assignedProblems.map(p => ({
            problem_id: p.problem_id,
            points_value: p.points_value || 100
          }))
        });

        // Seed available problems with assigned ones so they show up even if not in the first page
        setAssignedCache(assignedProblems);
        setAvailableProblems(assignedProblems);
      } catch (err) {
        console.error("Failed to fetch contest data", err);
      } finally {
        setLoading(false);
      }
    };

    loadContestData();
  }, [id]);

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

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({ ...prev, [name]: type === 'checkbox' ? checked : value }));
  };

  const handleArrayChange = (field, value) => {
    const arrayValue = value.split(',').map(item => item.trim()).filter(item => item !== '');
    const finalArray = field === 'allowed_graduation_years'
      ? arrayValue.map(v => parseInt(v, 10)).filter(v => !isNaN(v))
      : arrayValue;

    setFormData(prev => ({
      ...prev,
      access_rules: {
        ...(prev.access_rules || {}),
        [field]: finalArray
      }
    }));
  };

  // 👇 CRASH FIX 3: Safe array extraction for the UI inputs
  const getRuleValue = (field) => {
    const rules = formData.access_rules || {};
    const val = rules[field];
    return Array.isArray(val) ? val.join(', ') : '';
  };

  const toggleProblem = (prob) => {
    setFormData(prev => {
      const problems = Array.isArray(prev.problems) ? prev.problems : [];
      const exists = problems.find(p => p.problem_id === prob.problem_id);
      if (exists) {
        return { ...prev, problems: problems.filter(p => p.problem_id !== prob.problem_id) };
      } else {
        return { ...prev, problems: [...problems, { problem_id: prob.problem_id, points_value: 100 }] };
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
    setFormData(prev => {
      const problems = Array.isArray(prev.problems) ? prev.problems : [];
      return {
        ...prev,
        problems: problems.map(p => p.problem_id === problemId ? { ...p, points_value: parseInt(points, 10) || 0 } : p)
      };
    });
  };

  const handleSubmit = async () => {
    setSaving(true);
    try {
      const payload = {
        ...formData,
        start_time: new Date(formData.start_time).toISOString(),
        end_time: new Date(formData.end_time).toISOString(),
      };
      await api.put(`/contests/${id}`, payload);
      navigate('/contests');
    } catch (error) {
      console.error("Failed to update contest:", error);
      alert(`Backend Error: ${error.response?.data?.error || "Unknown Error. Check console."}`);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (window.confirm("CRITICAL WARNING: Are you sure you want to permanently destroy this contest? ALL contest history, including student submissions, live leaderboards, and anti-cheat telemetry, will be permanently deleted. Linked problems will safely remain in your repository.")) {
      setSaving(true);
      try {
        await api.delete(`/contests/${id}`);
        navigate('/contests');
      } catch (error) {
        console.error("Failed to delete contest", error);
        alert("Error deleting contest. Check console.");
        setSaving(false);
      }
    }
  };

  if (loading) return <div className="text-center py-20 text-gray-400 font-mono animate-pulse">Loading Arena Configurations...</div>;

  return (
    <div className="max-w-4xl mx-auto p-8">
      <div className="mb-8 border-b border-dark-border pb-4 flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold text-white">Configure Contest</h2>
          <p className="text-gray-400 mt-1">Update rules, timings, and problems for {formData.title}</p>
        </div>
        <div className="flex gap-2">
          {[1, 2, 3].map(num => (
            <div key={num} className={`h-2 w-12 rounded-full ${step >= num ? 'bg-blue-500' : 'bg-gray-700'}`} />
          ))}
        </div>
      </div>

      <div className="bg-[#1e1e1e] border border-dark-border rounded-lg p-6 shadow-xl">
        {step === 1 && (
          <div className="space-y-6 animate-fadeIn">
            <h3 className="text-xl font-bold text-white border-b border-dark-border pb-2">Step 1: Details & Timing</h3>
            <div className="grid grid-cols-2 gap-6">
              <div className="col-span-2">
                <label className="block text-gray-400 text-sm font-bold mb-2">Contest Title</label>
                <input type="text" name="title" value={formData.title} onChange={handleChange} className="w-full p-3 rounded bg-dark-surface border border-dark-border text-white focus:outline-none focus:border-blue-500" required />
              </div>
              <div className="col-span-2">
                <label className="block text-gray-400 text-sm font-bold mb-2">Host Organization</label>
                <input type="text" name="host_organization" value={formData.host_organization} onChange={handleChange} className="w-full p-3 rounded bg-dark-surface border border-dark-border text-white focus:outline-none focus:border-blue-500" />
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
                  <p className="text-xs text-gray-400">If unchecked, this contest will revert to a draft in your workspace.</p>
                </div>
              </div>
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-6 animate-fadeIn">
            <h3 className="text-xl font-bold text-white border-b border-dark-border pb-2">Step 2: Demographic Clearance</h3>
            <p className="text-sm text-gray-400 mb-4">Leave a field blank to allow everyone. Separate multiple entries with commas.</p>
            <div className="grid grid-cols-2 gap-6">
              {[
                { label: 'Allowed Courses', field: 'allowed_courses' },
                { label: 'Allowed Departments', field: 'allowed_departments' },
                { label: 'Allowed Batches', field: 'allowed_batches' },
                { label: 'Allowed Sections', field: 'allowed_sections' },
                { label: 'Allowed Student Groups', field: 'allowed_student_groups' },
                { label: 'Allowed Graduation Years', field: 'allowed_graduation_years' },
              ].map(item => (
                <div key={item.field}>
                  <label className="block text-gray-400 text-sm font-bold mb-2">{item.label}</label>
                  <input
                    type="text"
                    value={getRuleValue(item.field)}
                    onChange={(e) => handleArrayChange(item.field, e.target.value)}
                    className="w-full p-3 rounded bg-dark-surface border border-dark-border text-white focus:border-blue-500 outline-none font-mono text-sm"
                  />
                </div>
              ))}
            </div>
          </div>
        )}

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

            <h4 className="text-sm font-bold text-gray-400 mb-2">Problem Repository</h4>
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
                <p className="text-center text-gray-500 italic py-10">No problems found in your workspace.</p>
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

        <div className="flex justify-between mt-8 pt-6 border-t border-dark-border">
          {isTimeLocked ? (
            <div className="w-full text-center p-3 bg-red-900/20 border border-red-800/50 rounded text-red-400 font-bold">
              🔒 Time Lock Active: This contest has already started and cannot be modified.
            </div>
          ) : (
            <>
              <div className="flex gap-4">
                {step > 1 ? <Button onClick={() => setStep(step - 1)} variant="secondary">Back</Button> : <div></div>}
                {(userRole === 'admin' || (userRole === 'professor' && !formData.is_public)) && (
                  <Button onClick={handleDelete} variant="danger" disabled={saving} className="bg-red-900/50 border border-red-600 text-red-500 hover:bg-red-600 hover:text-white transition">
                    Delete Arena
                  </Button>
                )}
              </div>
              {step < 3 ? (
                <Button onClick={() => setStep(step + 1)} variant="primary">Next Step</Button>
              ) : (
                <Button onClick={handleSubmit} variant="success" disabled={saving} className="border border-green-600">
                  {saving ? 'Saving Arena...' : 'Update Contest'}
                </Button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}