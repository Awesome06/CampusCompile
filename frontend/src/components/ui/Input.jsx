import React from 'react';

export default function Input({ label, name, type = "text", value, onChange, placeholder, required, step }) {
  return (
    <div className="w-full">
      <label className="block text-sm font-bold text-gray-400 mb-2 uppercase tracking-wide">
        {label}
      </label>
      <input 
        type={type}
        name={name}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        required={required}
        step={step}
        className="w-full bg-[#2a2a2a] text-white p-3 rounded border border-dark-border outline-none focus:border-gray-500 transition"
      />
    </div>
  );
}