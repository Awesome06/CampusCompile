import React from 'react';

export default function Button({ children, onClick, type = "button", disabled, variant = "primary", size = "lg", className = "" }) {
  // Base styles (No width or padding here!)
  const baseStyle = "font-bold rounded shadow-lg transition tracking-wide flex items-center justify-center gap-2";
  
  const variants = {
    primary: "bg-blue-600 text-white hover:bg-blue-500",
    success: "bg-green-700 text-white hover:bg-green-600",
    danger: "bg-red-600 text-white hover:bg-red-500",
    microsoft: "bg-[#0078D4] text-white hover:bg-[#005ea6]",
    secondary: "bg-gray-700 text-white hover:bg-gray-600"
  };

  // Two strict size profiles
  const sizes = {
    lg: "w-full py-3 px-4 text-base", // Default: Huge forms (Onboarding, Login)
    sm: "w-auto py-1.5 px-4 text-sm"  // Small: Toolbars & Headers (Arena, ProblemList)
  };

  const disabledStyle = disabled ? "opacity-50 cursor-not-allowed bg-gray-600 hover:bg-gray-600" : "";

  return (
    <button 
      type={type} 
      onClick={onClick} 
      disabled={disabled}
      className={`${baseStyle} ${variants[variant]} ${sizes[size]} ${disabledStyle} ${className}`}
    >
      {children}
    </button>
  );
}