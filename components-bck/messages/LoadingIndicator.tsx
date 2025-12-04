
import React from 'react';

export const LoadingIndicator: React.FC = () => {
  return (
    <div className="flex justify-start">
      <div className="max-w-sm bg-gray-700 rounded-lg p-3 shadow flex items-center space-x-2">
        <div className="w-2 h-2 bg-cyan-400 rounded-full animate-bounce [animation-delay:-0.3s]"></div>
        <div className="w-2 h-2 bg-cyan-400 rounded-full animate-bounce [animation-delay:-0.15s]"></div>
        <div className="w-2 h-2 bg-cyan-400 rounded-full animate-bounce"></div>
      </div>
    </div>
  );
};
