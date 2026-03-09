import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import api from '../services/api';
import Button from '../components/ui/Button';

export default function EditContest() {
  const { id } = useParams();
  const navigate = useNavigate();
  const userRole = localStorage.getItem('role');
  const isTimeLocked = formData.start_time && new Date(formData.start_time) <= new Date() && userRole !== 'admin';
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [availableProblems, setAvailableProblems] = useState([]);

  const defaultAccessRules = {
    allowed_courses: [], allowed_departments: [], allowed_batches: [],
    allowed_sections: [], allowed_student_groups: [], allowed_graduation_years: []
  };

  const [formData, setFormData] = useState({
    title: '',
    host_organization: '',
    start_time: '',
    end_time: '',
    is_public: false,
    access_rules: defaultAccessRules,
    problems: []
  });

  // Helper to format ISO date strings for <input type="datetime-local">
  const formatForInput = (isoString) => {
    if (!isoString) return '';
    try {
      return new Date(isoString).toISOString().slice(0, 16);
    } catch (e) {
      console.warn("Invalid date format received:", isoString);
      return ''; // Fallback gracefully instead of crashing the page
    }
  };

  useEffect(() => {
    // Fetch the contest, its currently assigned problems, and the professor's workspace problems
    Promise.all([
      api.get(`/contests/${id}`),
      api.get(`/contests/${id}/problems`),
      api.get('/faculty/problems')
    ])
    .then(([contestRes, assignedProbsRes, availableProbsRes]) => {
      const contest = contestRes.data.contest;
      const assignedProblems = assignedProbsRes.data || [];
      
      setAvailableProblems(availableProbsRes.data || []);
      
      setFormData({
        title: contest.title || '',
        host_organization: contest.host_organization || 'CampusCompile Official',
        start_time: formatForInput(contest.start_time),
        end_time: formatForInput(contest.end_time),
        is_public: contest.is_public || false,
        access_rules: contest.access_rules || defaultAccessRules,
        // Map the backend structure to our frontend checklist structure
        problems: assignedProblems.map(p => ({
          problem_id: p.problem_id,
          points_value: p.points_value
        }))
      });
      setLoading(false);
    })
    .catch(err => {
      console.error("Failed to fetch contest data", err);
      alert("Error loading contest details.");
      setLoading(false);
    });
  }, [id]);

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const handleArrayChange = (field, value) => {
    const arrayValue = value.split(',').map(item => item.trim()).filter(item => item !== '');
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

  const toggleProblem = (problemId) => {
    setFormData(prev => {
      const exists = prev.problems.find(p => p.problem_id === problemId);
      if (exists) {
        return { ...prev, problems: prev.problems.filter(p => p.problem_id !== problemId) };
      } else {
        return { ...prev, problems: [...prev.problems, { problem_id: problemId, points_value: 100 }] };
      }
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
    setSaving(true);
    try {
      // Create a cloned payload and force the dates into full ISO strings for Go
      const payload = {
        ...formData,
        start_time: new Date(formData.start_time).toISOString(),
        end_time: new Date(formData.end_time).toISOString(),
      };

      // Send the updated payload to the Go backend via PUT
      await api.put(`/contests/${id}`, payload);
      navigate('/contests');
    } catch (error) {
      // Log the EXACT error message the Go backend sends back
      console.error("Failed to update contest:", error.response?.data || error.message);
      
      // Show the exact Gin validation error in the alert
      const errorMsg = error.response?.data?.error || "Unknown Error. Check console.";
      alert(`Backend Error: ${errorMsg}`);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (window.confirm("CRITICAL WARNING: Are you sure you want to permanently destroy this contest? Student submissions will be preserved as unranked practice runs.")) {
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

  if (loading) {
    return <div className="text-center py-20 text-gray-400 font-mono animate-pulse">Loading Arena Configurations...</div>;
  }

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
        
        {/* === STEP 1: Details === */}
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

        {/* === STEP 2: Access Control === */}
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
                    defaultValue={((formData.access_rules || {})[item.field] || []).join(', ')}
                    onChange={(e) => handleArrayChange(item.field, e.target.value)} 
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

            <div className="max-h-96 overflow-y-auto pr-2 space-y-3">
              {availableProblems.length === 0 ? (
                <p className="text-center text-gray-500 italic py-10">No problems found in your workspace.</p>
              ) : (
                availableProblems.map(prob => {
                  const isSelected = formData.problems.some(p => p.problem_id === prob.problem_id);
                  const selectedData = formData.problems.find(p => p.problem_id === prob.problem_id);
                  
                  return (
                    <div key={prob.problem_id} className={`flex items-center justify-between p-4 rounded border transition ${isSelected ? 'border-blue-500 bg-blue-900/10' : 'border-dark-border bg-dark-surface hover:bg-[#252525]'}`}>
                      <div className="flex items-center gap-4">
                        <input type="checkbox" checked={isSelected} onChange={() => toggleProblem(prob.problem_id)} className="w-5 h-5 accent-blue-500" />
                        <div>
                          <p className="text-white font-bold">{prob.title}</p>
                          <p className={`text-xs ${prob.difficulty === 'Easy' ? 'text-green-400' : prob.difficulty === 'Medium' ? 'text-yellow-400' : 'text-red-400'}`}>{prob.difficulty}</p>
                        </div>
                      </div>
                      
                      {isSelected && (
                        <div className="flex items-center gap-2">
                          <label className="text-sm text-gray-400 font-bold">Points:</label>
                          <input 
                            type="number" 
                            value={selectedData?.points_value || 100} 
                            onChange={(e) => updatePoints(prob.problem_id, e.target.value)} 
                            className="w-20 p-1.5 rounded bg-[#1e1e1e] border border-dark-border text-white text-center font-mono"
                          />
                        </div>
                      )}
                    </div>
                  )
                })
              )}
            </div>
          </div>
        )}

        {/* Form Navigation Controls */}
          <div className="flex justify-between mt-8 pt-6 border-t border-dark-border">
          
            {/* If Time Locked, show a warning instead of buttons */}
            {isTimeLocked ? (
              <div className="w-full text-center p-3 bg-red-900/20 border border-red-800/50 rounded text-red-400 font-bold">
                🔒 Time Lock Active: This contest has already started and cannot be modified.
              </div>
            ) : (
              <>
                <div className="flex gap-4">
                  {step > 1 ? (
                    <Button onClick={() => setStep(step - 1)} variant="secondary">Back</Button>
                  ) : <div></div>}

                  {userRole === 'admin' && (
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