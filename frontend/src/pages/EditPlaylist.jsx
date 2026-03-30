import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import api from '../services/api';
import Button from '../components/ui/Button';

const PLAYLIST_TAGS = [
  'Array', 'Hash Table', 'String', 'Two Pointers', 'Sliding Window',
  'Matrix', 'Linked List', 'Math', 'Greedy', 'Binary Tree',
  'Binary Search Tree', 'Graph', 'Trie', 'Backtracking',
  'Dynamic Programming', 'Bit Manipulation', 'Divide and Conquer', 'Sorting'
];

export default function EditPlaylist() {
  const navigate = useNavigate();
  const { id } = useParams();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [isPublic, setIsPublic] = useState(false);
  const [overallDifficulty, setOverallDifficulty] = useState('Medium');
  const [selectedTags, setSelectedTags] = useState([]);

  // Problem Selection
  const [availableProblems, setAvailableProblems] = useState([]);
  const [selectedProblems, setSelectedProblems] = useState([]); // [{ problem_id, title, difficulty, custom_difficulty }]

  // Problem Search modal
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    // Fetch all public/faculty problems
    api.get('/faculty/problems', { params: { limit: 50, offset: 0 } })
      .then((res) => {
        const data = Array.isArray(res.data) ? res.data : (res.data?.problems || []);
        setAvailableProblems(data);
      })
      .catch((err) => console.error("Error fetching problems:", err));

    // Fetch Target Playlist
    api.get(`/playlists/${id}`).then(res => {
      const p = res.data;
      setTitle(p.title);
      setDescription(p.description || '');
      setIsPublic(p.is_public);
      setOverallDifficulty(p.overall_difficulty);
      setSelectedTags(p.tags || []);
    }).catch(err => {
      setError("Failed to fetch existing playlist configuration.");
    });

    // Fetch Sequence
    api.get(`/playlists/${id}/problems`).then(res => {
      setSelectedProblems(res.data.map(prob => ({
        problem_id: prob.problem_id,
        title: prob.title,
        difficulty: prob.difficulty,
        custom_difficulty: prob.custom_difficulty || 'No Change',
        order_index: prob.order_index
      })));
    }).catch(err => console.error(err));

  }, [id]);

  const handleFetchMoreProblems = () => {
    api.get('/faculty/problems', { params: { limit: 50, search: searchQuery } })
      .then((res) => {
        const data = Array.isArray(res.data) ? res.data : (res.data?.problems || []);
        setAvailableProblems(data);
      })
      .catch((err) => console.error(err));
  };

  const handleAddProblem = (prob) => {
    if (!selectedProblems.find(p => p.problem_id === prob.problem_id)) {
      setSelectedProblems([
        ...selectedProblems,
        { problem_id: prob.problem_id, title: prob.title, difficulty: prob.difficulty, custom_difficulty: 'No Change' }
      ]);
    }
  };

  const handleRemoveProblem = (problemId) => {
    setSelectedProblems(selectedProblems.filter(p => p.problem_id !== problemId));
  };

  const handleCustomDiffChange = (problemId, customDifficulty) => {
    setSelectedProblems(selectedProblems.map(p =>
      p.problem_id === problemId ? { ...p, custom_difficulty: customDifficulty } : p
    ));
  };

  const toggleTag = (tag) => {
    if (selectedTags.includes(tag)) {
      setSelectedTags(selectedTags.filter(t => t !== tag));
    } else {
      setSelectedTags([...selectedTags, tag]);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!title) {
      setError("Title is required");
      return;
    }
    if (selectedProblems.length === 0) {
      setError("You must select at least one problem");
      return;
    }

    setLoading(true);
    setError(null);

    const payload = {
      title,
      description,
      is_public: isPublic,
      overall_difficulty: overallDifficulty,
      tags: selectedTags,
      problems: selectedProblems.map(p => ({
        problem_id: p.problem_id,
        custom_difficulty: p.custom_difficulty === 'No Change' ? null : (p.custom_difficulty || null)
      }))
    };

    try {
      await api.put(`/playlists/${id}`, payload);
      navigate(`/playlists/${id}`);
    } catch (err) {
      setError(err.response?.data?.error || "Failed to update/save playlist");
      setLoading(false);
    }
  };

  return (
    <div className="p-8 max-w-4xl mx-auto text-white">
      <h1 className="text-3xl font-bold mb-8">Edit Playlist Sequence</h1>

      {error && (
        <div className="bg-red-900/40 border border-red-500 text-red-300 p-4 rounded mb-6 font-mono text-sm leading-relaxed whitespace-pre-wrap">
          <p className="font-bold border-b border-red-500/50 mb-2 pb-1">Error Updating Playlist Sequence</p>
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6 bg-[#1e1e1e] p-6 rounded-lg border border-dark-border shadow-lg">
        <div>
          <label className="block text-sm font-medium text-gray-400 mb-1">Playlist Title *</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
            className="w-full bg-dark-bg border border-dark-border rounded px-4 py-2 focus:border-blue-500 outline-none text-white shadow-inner block"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-400 mb-1">Description</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full bg-dark-bg border border-dark-border rounded px-4 py-2 focus:border-blue-500 outline-none text-white shadow-inner min-h-[100px] block"
          />
        </div>

        <div className="flex gap-4">
          <div className="w-1/2">
            <label className="block text-sm font-medium text-gray-400 mb-1">Overall Difficulty</label>
            <select
              value={overallDifficulty}
              onChange={(e) => setOverallDifficulty(e.target.value)}
              className="w-full bg-dark-bg border border-dark-border rounded px-4 py-2 text-white focus:outline-none focus:border-blue-500"
            >
              <option value="Easy">Easy</option>
              <option value="Medium">Medium</option>
              <option value="Hard">Hard</option>
            </select>
          </div>
          <div className="w-1/2 flex items-center pt-6">
            <label className="flex items-center gap-3 cursor-pointer group">
              <input
                type="checkbox"
                checked={isPublic}
                onChange={(e) => setIsPublic(e.target.checked)}
                className="w-5 h-5 rounded border-gray-600 text-blue-600 focus:ring-blue-500 focus:ring-offset-gray-900 bg-gray-800 transition"
              />
              <span className="text-gray-300 font-medium group-hover:text-white transition">Publish Playlist (Make visible to Students)</span>
            </label>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-400 mb-2">Category Tags</label>
          <div className="flex flex-wrap gap-2">
            {PLAYLIST_TAGS.map(tag => (
              <button
                key={tag}
                type="button"
                onClick={() => toggleTag(tag)}
                className={`px-3 py-1 rounded-full text-xs font-mono border transition ${selectedTags.includes(tag)
                  ? 'bg-blue-600/20 text-blue-400 border-blue-500/50 hover:bg-blue-600/30'
                  : 'bg-dark-surface text-gray-400 border-dark-border hover:bg-gray-700 hover:text-white'
                  }`}
              >
                {tag}
              </button>
            ))}
          </div>
        </div>

        <hr className="border-dark-border my-8" />

        {/* Selected Problems Configuration */}
        <div>
          <h2 className="text-xl font-bold mb-4">Problem Sequence</h2>
          <p className="text-sm text-gray-400 mb-4">Edit the problem sequence mapping below.</p>

          {selectedProblems.length === 0 ? (
            <div className="bg-[#2a2a2a] p-4 text-center rounded border border-dashed border-gray-600 text-gray-500 font-mono text-sm">
              Your playlist is empty. Add problems below.
            </div>
          ) : (
            <div className="space-y-3">
              {selectedProblems.map((prob, idx) => (
                <div key={prob.problem_id} className="flex gap-4 items-center bg-[#2a2a2a] p-3 rounded border border-dark-border shadow-sm">
                  <span className="text-gray-500 font-bold w-6">{idx + 1}.</span>
                  <div className="flex-1">
                    <div className="font-bold text-gray-200">{prob.title}</div>
                    <div className="text-xs text-gray-400 font-mono">Standard Diff: {prob.difficulty}</div>
                  </div>
                  <div>
                    <select
                      value={prob.custom_difficulty || 'No Change'}
                      onChange={(e) => handleCustomDiffChange(prob.problem_id, e.target.value)}
                      className="w-36 bg-dark-bg border border-dark-border rounded px-2 py-1 text-sm focus:border-blue-500 focus:outline-none text-white shadow-inner"
                    >
                      <option value="No Change">No Change</option>
                      <option value="Easy">Easy</option>
                      <option value="Medium">Medium</option>
                      <option value="Hard">Hard</option>
                    </select>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleRemoveProblem(prob.problem_id)}
                    className="text-red-400 hover:text-red-300 mx-2 transition"
                  >
                    🗑️
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <hr className="border-dark-border my-6" />

        {/* Problem Search Interface */}
        <div className="bg-[#2a2a2a] p-4 rounded border border-dark-border">
          <h3 className="font-bold text-lg mb-3">Find Problems</h3>
          <div className="flex gap-2 mb-4">
            <input
              type="text"
              placeholder="Search problems..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="flex-1 bg-dark-bg border border-dark-border rounded px-4 py-2 text-sm focus:border-blue-500 focus:outline-none text-white"
            />
            <Button type="button" onClick={handleFetchMoreProblems} variant="secondary">Search</Button>
          </div>

          <div className="max-h-60 overflow-y-auto pr-2 custom-scrollbar space-y-2">
            {availableProblems.length === 0 && (
              <p className="text-gray-500 text-sm text-center py-2">No problems match your query.</p>
            )}
            {availableProblems.map(prob => {
              const isSelected = selectedProblems.some(p => p.problem_id === prob.problem_id);
              return (
                <div key={prob.problem_id} className="flex justify-between items-center p-2 hover:bg-[#333] rounded">
                  <span className="text-sm font-medium">{prob.title}</span>
                  <div className="flex items-center gap-4">
                    <span className="text-xs text-gray-500 border border-gray-600 px-1 py-0.5 rounded uppercase">{prob.difficulty}</span>
                    <button
                      type="button"
                      disabled={isSelected}
                      onClick={() => handleAddProblem(prob)}
                      className={`px-3 py-1 text-xs font-bold rounded min-w-[60px] ${isSelected
                        ? 'bg-gray-700 text-gray-500 cursor-not-allowed border-gray-600'
                        : 'bg-green-600/20 text-green-400 border border-green-600/50 hover:bg-green-600/40 transition'
                        }`}
                    >
                      {isSelected ? 'Added' : 'Add'}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        <div className="pt-4 border-t border-dark-border mt-6">
          <Button type="submit" disabled={loading} variant="primary" className="w-full border-blue-600 shadow-md h-12">
            {loading ? 'Saving Changes...' : 'Save Playlist Configuration'}
          </Button>
        </div>
      </form>
    </div>
  );
}
