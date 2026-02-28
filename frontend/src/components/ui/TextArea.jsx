import React from 'react';

export default function TextArea({ label, name, value, onChange, placeholder, required, rows = 4 }) {
  return (
    <div className="w-full">
      <label className="block text-sm font-bold text-gray-400 mb-2 uppercase tracking-wide">
        {label}
      </label>
      <textarea 
        name={name}
        required={required}
        rows={rows}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        className="w-full bg-[#2a2a2a] text-white p-3 rounded border border-dark-border outline-none focus:border-gray-500 transition resize-none font-mono text-sm"
      />
    </div>
  );
}