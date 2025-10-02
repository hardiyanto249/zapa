
import React from 'react';

interface UserMessageProps {
  text: string;
}

export const UserMessage: React.FC<UserMessageProps> = ({ text }) => {
  return (
    <div className="flex justify-end">
      <div className="max-w-md lg:max-w-2xl bg-cyan-600 rounded-lg p-3 shadow">
        <p className="text-white whitespace-pre-wrap">{text}</p>
      </div>
    </div>
  );
};
