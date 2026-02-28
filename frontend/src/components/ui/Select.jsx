import React from 'react';

export default function Select({ label, name, value, onChange, options }) {
  return (
    <div className="w-full">
      <label className="block text-sm font-bold text-gray-400 mb-2 uppercase tracking-wide">
        {label}
      </label>
      <select 
        name={name}
        value={value}
        onChange={onChange}
        className="w-full bg-[#2a2a2a] text-white p-3 rounded border border-dark-border outline-none cursor-pointer focus:border-gray-500 transition"
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  );
}