import React from 'react';

export const markdownComponents = {
  h3: ({node, ...props}) => <h3 className="text-lg font-bold text-blue-400 mt-8 mb-4 border-b border-dark-border pb-2 tracking-wide uppercase" {...props} />,
  pre: ({node, ...props}) => <pre className="bg-[#121212] border border-dark-border rounded-lg p-5 overflow-x-auto my-4 font-mono text-sm text-gray-300 shadow-inner custom-scrollbar" {...props} />,
  code: ({node, className, children, ...props}) => {
    const isInline = !className || !className.includes('language-');
    return isInline 
      ? <code className="bg-[#2a2a2a] text-pink-400 px-1.5 py-0.5 rounded text-sm font-mono border border-dark-border" {...props}>{children}</code> 
      : <code className={className} {...props}>{children}</code>;
  },
  ul: ({node, ...props}) => <ul className="list-disc list-inside my-4 space-y-2 text-gray-300 marker:text-blue-500" {...props} />,
  li: ({node, ...props}) => <li className="leading-relaxed" {...props} />,
  p: ({node, ...props}) => <p className="my-4 leading-relaxed text-gray-300" {...props} />,
  blockquote: ({node, ...props}) => <blockquote className="border-l-4 border-blue-500 bg-blue-900/10 p-4 my-4 rounded-r-lg italic text-gray-400" {...props} />
};